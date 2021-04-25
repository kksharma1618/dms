package dms

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/kksharma1618/dms/dlna"
	"github.com/kksharma1618/dms/misc"
	"github.com/kksharma1618/dms/upnp"
	"github.com/kksharma1618/dms/upnpav"
)

type contentProviderServerItem struct {
	ID           string `json:"id"`
	ParentID     string `json:"parent_id"`
	IsDirectory  bool   `json:"is_directory"`
	Title        string `json:"title"`
	MimeType     string `json:"mime_type,omitempty"`
	MediaURL     string `json:"media_url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Size         uint64 `json:"size,omitempty"`
	Bitrate      uint   `json:"bitrate,omitempty"`
	Duration     int    `json:"duration,omitempty"`
	Resolution   string `json:"resolution,omitempty"`
}

var apiClient *http.Client = nil
var apiTransport *http.Transport = nil

func (me *Server) serveCdpProxy(res http.ResponseWriter, req *http.Request) {
	target := req.URL.Query().Get("url")
	// parse the url
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.Transport = me.getApiTransport()

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

func (me *Server) getApiTransport() *http.Transport {
	if apiTransport != nil {
		return apiTransport
	}
	if me.ContentProviderServerRootCas == "" {
		apiTransport = &http.Transport{}
		return apiTransport
	}

	rootCasFiles := strings.Split(me.ContentProviderServerRootCas, ",")
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	for _, rootCasFile := range rootCasFiles {
		// Read in the cert file
		certs, err := ioutil.ReadFile(rootCasFile)
		if err != nil {
			log.Fatalf("Failed to append %q to RootCAs: %v", rootCasFile, err)
		}

		// Append our cert to the system pool
		if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
			log.Println("No certs appended, using system certs only")
		}
	}
	config := &tls.Config{
		RootCAs: rootCAs,
	}
	apiTransport = &http.Transport{TLSClientConfig: config}
	return apiTransport
}

func (me *Server) getApiClient() *http.Client {
	if apiClient != nil {
		return apiClient
	}
	apiClient = &http.Client{
		Transport: me.getApiTransport(),
		Timeout:   time.Second * 10, // Timeout after 2 seconds
	}
	return apiClient
}

func (me *contentDirectoryService) makeContentProviderApiRequest(path string) (body []byte, err error) {
	apiClient := me.getApiClient()
	req, err := http.NewRequest(http.MethodGet, me.ContentProviderServer+path, nil)
	if err != nil {
		fmt.Println("api.err", err)
		return
	}
	req.Header.Set("Authorization", "Bearer: "+me.ContentProviderServerToken)

	res, err := apiClient.Do(req)
	if err != nil {
		fmt.Println("api.err", err)
		return
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err = ioutil.ReadAll(res.Body)
	return
}

// Turns the given entry and DMS host into a UPnP object. A nil object is
// returned if the entry is not of interest.
func (me *contentDirectoryService) contentProviderObjectToUpnpObject(cdpObject contentProviderServerItem, host, userAgent string) (ret interface{}, err error) {

	obj := upnpav.Object{
		ID:         cdpObject.ID,
		Restricted: 1,
		ParentID:   cdpObject.ParentID,
		Title:      cdpObject.Title,
	}
	if cdpObject.IsDirectory {
		obj.Class = "object.container.storageFolder"
		obj.Title = cdpObject.Title
		ret = upnpav.Container{Object: obj}
		return
	}
	if cdpObject.MediaURL == "" {
		return
	}
	if cdpObject.ThumbnailURL != "" {
		iconURI := (&url.URL{
			Scheme: "http",
			Host:   host,
			Path:   cdpProxyPath,
			RawQuery: url.Values{
				"url": {cdpObject.ThumbnailURL},
			}.Encode(),
		}).String()
		obj.Icon = iconURI
		obj.AlbumArtURI = iconURI
	}
	mtype := mimeType(cdpObject.MimeType)
	if !mtype.IsMedia() {
		return
	}
	obj.Class = "object.item." + mtype.Type() + "Item"
	obj.Title = cdpObject.Title
	item := upnpav.Item{
		Object: obj,
		// Capacity: 1 for raw, 1 for icon, plus transcodes.
		Res: make([]upnpav.Resource, 0, 2+len(transcodes)),
	}
	item.Res = append(item.Res, upnpav.Resource{
		URL: (&url.URL{
			Scheme: "http",
			Host:   host,
			Path:   cdpProxyPath,
			RawQuery: url.Values{
				"url": {cdpObject.MediaURL},
			}.Encode(),
		}).String(),
		ProtocolInfo: fmt.Sprintf("http-get:*:%s:%s", mtype, dlna.ContentFeatures{
			SupportRange: true,
		}.String()),
		Bitrate:    cdpObject.Bitrate,
		Duration:   misc.FormatDurationSexagesimal(time.Duration(cdpObject.Duration * 1000000000)),
		Size:       uint64(cdpObject.Size),
		Resolution: cdpObject.Resolution,
	})
	if obj.Icon != "" && (mtype.IsVideo() || mtype.IsImage()) {
		item.Res = append(item.Res, upnpav.Resource{
			URL:          obj.Icon,
			ProtocolInfo: "http-get:*:image/jpeg:DLNA.ORG_PN=JPEG_TN",
		})
	}
	ret = item
	return
}

func (me *contentDirectoryService) handleContentProviderServerBrowse(action string, argsXML []byte, r *http.Request) (map[string]string, error) {
	host := r.Host
	userAgent := r.UserAgent()
	var browse browse
	if err := xml.Unmarshal([]byte(argsXML), &browse); err != nil {
		return nil, err
	}
	switch browse.BrowseFlag {
	case "BrowseDirectChildren":
		body, err := me.makeContentProviderApiRequest("/browse?" + url.Values{
			"id": {browse.ObjectID},
		}.Encode())
		if err != nil {
			fmt.Println("BrowseDirectChildren.err", err)
			return nil, upnp.Errorf(upnpav.NoSuchObjectErrorCode, err.Error())
		}
		cdObjs := []contentProviderServerItem{}
		if err := json.Unmarshal(body, &cdObjs); err != nil {
			fmt.Println("BrowseDirectChildren.marshal.err", err)
			return nil, err
		}
		totalMatches := len(cdObjs)
		objs := make([]interface{}, 0, totalMatches)
		for _, cdObj := range cdObjs {
			obj, err := me.contentProviderObjectToUpnpObject(cdObj, host, userAgent)
			if err == nil {
				objs = append(objs, obj)
			}
		}
		objs = objs[func() (low int) {
			low = browse.StartingIndex
			if low > len(objs) {
				low = len(objs)
			}
			return
		}():]
		if browse.RequestedCount != 0 && int(browse.RequestedCount) < len(objs) {
			objs = objs[:browse.RequestedCount]
		}
		result, err := xml.Marshal(objs)
		if err != nil {
			return nil, err
		}
		return map[string]string{
			"TotalMatches":   fmt.Sprint(totalMatches),
			"NumberReturned": fmt.Sprint(len(objs)),
			"Result":         didl_lite(string(result)),
			"UpdateID":       me.updateIDString(),
		}, nil
	case "BrowseMetadata":
		// fileInfo, err := os.Stat(obj.FilePath())
		// if err != nil {
		// 	if os.IsNotExist(err) {
		// 		return nil, &upnp.Error{
		// 			Code: upnpav.NoSuchObjectErrorCode,
		// 			Desc: err.Error(),
		// 		}
		// 	}
		// 	return nil, err
		// }
		// upnp, err := me.cdsObjectToUpnpavObject(obj, fileInfo, host, userAgent)
		// if err != nil {
		// 	return nil, err
		// }
		// buf, err := xml.Marshal(upnp)
		// if err != nil {
		// 	return nil, err
		// }
		// return map[string]string{
		// 	"TotalMatches":   "1",
		// 	"NumberReturned": "1",
		// 	"Result":         didl_lite(func() string { return string(buf) }()),
		// 	"UpdateID":       me.updateIDString(),
		// }, nil
	default:
		return nil, upnp.Errorf(upnp.ArgumentValueInvalidErrorCode, "unhandled browse flag: %v", browse.BrowseFlag)
	}
	return nil, nil
}
