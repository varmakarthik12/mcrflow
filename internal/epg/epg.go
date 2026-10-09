package epg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

// XMLTV models for standards-compliant EPG export.
type XMLTVRoot struct {
	XMLName           xml.Name         `xml:"tv"`
	GeneratorInfoName string           `xml:"generator-info-name,attr"`
	Channels          []XMLTVChannel   `xml:"channel"`
	Programmes        []XMLTVProgramme `xml:"programme"`
}

type XMLTVChannel struct {
	ID          string `xml:"id,attr"`
	DisplayName string `xml:"display-name"`
	Icon        *struct {
		Src string `xml:"src,attr"`
	} `xml:"icon,omitempty"`
}

type XMLTVTitle struct {
	Lang  string `xml:"lang,attr,omitempty"`
	Value string `xml:",chardata"`
}

type XMLTVDesc struct {
	Lang  string `xml:"lang,attr,omitempty"`
	Value string `xml:",chardata"`
}

type XMLTVRating struct {
	System string `xml:"system,attr,omitempty"`
	Value  string `xml:"value"`
}

type XMLTVProgramme struct {
	Start    string       `xml:"start,attr"`
	Stop     string       `xml:"stop,attr"`
	Channel  string       `xml:"channel,attr"`
	Titles   []XMLTVTitle `xml:"title"`
	Descs    []XMLTVDesc  `xml:"desc,omitempty"`
	Category string       `xml:"category,omitempty"`
	Rating   *XMLTVRating `xml:"rating,omitempty"`
	Icon     *struct {
		Src string `xml:"src,attr"`
	} `xml:"icon,omitempty"`
}

// FormatXmltvTime formats time as XMLTV timestamp: YYYYMMDDHHMMSS +0530
func FormatXmltvTime(t time.Time) string {
	// Format in IST (+0530)
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err == nil {
		t = t.In(loc)
	}
	return t.Format("20060102150405 -0700")
}

// GenerateXMLTV creates an XMLTV XML feed for given channel and schedule items.
func GenerateXMLTV(channel *models.Channel, items []*models.ScheduleItem) ([]byte, error) {
	root := XMLTVRoot{
		GeneratorInfoName: "MCRFlow Master Control Playout",
		Channels: []XMLTVChannel{
			{
				ID:          channel.ID,
				DisplayName: channel.Name,
			},
		},
	}

	if channel.LogoURL != "" {
		root.Channels[0].Icon = &struct {
			Src string `xml:"src,attr"`
		}{Src: channel.LogoURL}
	}

	for _, item := range items {
		p := XMLTVProgramme{
			Start:   FormatXmltvTime(item.StartTime),
			Stop:    FormatXmltvTime(item.EndTime),
			Channel: channel.ID,
		}

		// Titles & Metadata
		if item.TmdbMetadata != nil {
			if item.TmdbMetadata.LocalizedTitle != "" {
				p.Titles = append(p.Titles, XMLTVTitle{Lang: "hi", Value: item.TmdbMetadata.LocalizedTitle})
			}
			p.Titles = append(p.Titles, XMLTVTitle{Lang: "en", Value: item.TmdbMetadata.Title})

			if item.TmdbMetadata.Overview != "" {
				p.Descs = append(p.Descs, XMLTVDesc{Lang: "en", Value: item.TmdbMetadata.Overview})
			}

			if len(item.TmdbMetadata.Genres) > 0 {
				p.Category = strings.Join(item.TmdbMetadata.Genres, ", ")
			}

			if item.TmdbMetadata.ContentRating != "" {
				p.Rating = &XMLTVRating{
					System: "CBFC",
					Value:  item.TmdbMetadata.ContentRating,
				}
			}

			if item.TmdbMetadata.PosterURL != "" {
				p.Icon = &struct {
					Src string `xml:"src,attr"`
				}{Src: item.TmdbMetadata.PosterURL}
			}
		} else {
			p.Titles = append(p.Titles, XMLTVTitle{Lang: "en", Value: "Broadcast Program"})
		}

		root.Programmes = append(root.Programmes, p)
	}

	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	if err := encoder.Encode(root); err != nil {
		return nil, fmt.Errorf("failed to encode XMLTV: %w", err)
	}

	return buf.Bytes(), nil
}
