package upnpav

import (
	"encoding/xml"
)

const (
	NoSuchObjectErrorCode = 701
)

type Resource struct {
	XMLName      xml.Name `xml:"res" json:"res"`
	ProtocolInfo string   `xml:"protocolInfo,attr" json:"protocolInfo"`
	URL          string   `xml:",chardata" json:"url"`
	Size         uint64   `xml:"size,attr,omitempty" json:"size,omitempty"`
	Bitrate      uint     `xml:"bitrate,attr,omitempty" json:"bitrate,omitempty"`
	Duration     string   `xml:"duration,attr,omitempty" json:"duration,omitempty"`
	Resolution   string   `xml:"resolution,attr,omitempty" json:"resolution,omitempty"`
}

type Container struct {
	Object
	XMLName    xml.Name `xml:"container" json:"container"`
	ChildCount int      `xml:"childCount,attr" json:"childCount"`
}

type Item struct {
	Object
	XMLName xml.Name `xml:"item" json:"item"`
	Res     []Resource
}

type Object struct {
	ID          string `xml:"id,attr" json:"id"`
	ParentID    string `xml:"parentID,attr" json:"parentID"`
	Restricted  int    `xml:"restricted,attr" json:"restricted"` // indicates whether the object is modifiable
	Class       string `xml:"upnp:class" json:"upnp:class"`
	Icon        string `xml:"upnp:icon,omitempty" json:"upnp:icon,omitempty"`
	Title       string `xml:"dc:title" json:"dc:title"`
	Artist      string `xml:"upnp:artist,omitempty" json:"upnp:artist,omitempty"`
	Album       string `xml:"upnp:album,omitempty" json:"upnp:album,omitempty"`
	Genre       string `xml:"upnp:genre,omitempty" json:"upnp:genre,omitempty"`
	AlbumArtURI string `xml:"upnp:albumArtURI,omitempty" json:"upnp:albumArtURI,omitempty"`
	Searchable  int    `xml:"searchable,attr" json:"searchable"`
}
