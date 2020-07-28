package dms

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"

	"github.com/anacrolix/dms/dlna"
	"github.com/anacrolix/dms/upnp"
	"github.com/anacrolix/dms/upnpav"
)

type ContentProviderServerItem struct {
	ID           string `json:"id"`
	ParentID     string `json:"parent_id"`
	IsDirectory  bool   `json:"is_directory"`
	Title        string `json:"title"`
	MimeType     string `json:"mime_type,omitempty"`
	MediaURL     string `json:"media_url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Size         uint64 `json:"size,omitempty"`
	Bitrate      uint   `json:"bitrate,omitempty"`
	Duration     string `json:"duration,omitempty"`
	Resolution   string `json:"resolution,omitempty"`
}

// Turns the given entry and DMS host into a UPnP object. A nil object is
// returned if the entry is not of interest.
func (me *contentDirectoryService) contentProviderObjectToUpnpObject(cdpObject ContentProviderServerItem, host, userAgent string) (ret interface{}, err error) {
	if cdpObject.MediaURL == "" {
		return
	}
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
		Duration:   cdpObject.Duration,
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
	var browse browse
	if err := xml.Unmarshal([]byte(argsXML), &browse); err != nil {
		return nil, err
	}
	obj, err := me.objectFromID(browse.ObjectID)
	if err != nil {
		return nil, upnp.Errorf(upnpav.NoSuchObjectErrorCode, err.Error())
	}
	fmt.Println(obj, err)
	switch browse.BrowseFlag {
	case "BrowseDirectChildren":
		// objs, err := me.readContainer(obj, host, userAgent)
		// if err != nil {
		// 	return nil, upnp.Errorf(upnpav.NoSuchObjectErrorCode, err.Error())
		// }
		// totalMatches := len(objs)
		// objs = objs[func() (low int) {
		// 	low = browse.StartingIndex
		// 	if low > len(objs) {
		// 		low = len(objs)
		// 	}
		// 	return
		// }():]
		// if browse.RequestedCount != 0 && int(browse.RequestedCount) < len(objs) {
		// 	objs = objs[:browse.RequestedCount]
		// }
		// fmt.Println(objs)
		// result, err := xml.Marshal(objs)
		// if err != nil {
		// 	return nil, err
		// }
		// return map[string]string{
		// 	"TotalMatches":   fmt.Sprint(totalMatches),
		// 	"NumberReturned": fmt.Sprint(len(objs)),
		// 	"Result":         didl_lite(string(result)),
		// 	"UpdateID":       me.updateIDString(),
		// }, nil
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
		// fmt.Println(upnp)
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
