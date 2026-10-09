package epg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/varmakarthik12/mcrflow/internal/models"
)

// XMLTV Schema structures
type XMLTVRoot struct {
	XMLName           xml.Name       `xml:"tv"`
	GeneratorInfoName string         `xml:"generator-info-name,attr"`
	Channels          []XMLTVChannel `xml:"channel"`
	Programmes        []XMLTVProgram `xml:"programme"`
}

type XMLTVChannel struct {
	ID          string     `xml:"id,attr"`
	DisplayName string     `xml:"display-name"`
	Icon        *XMLTVIcon `xml:"icon,omitempty"`
}

type XMLTVIcon struct {
	Src string `xml:"src,attr"`
}

type XMLTVText struct {
	Lang  string `xml:"lang,attr,omitempty"`
	Value string `xml:",chardata"`
}

type XMLTVProgram struct {
	Start   string     `xml:"start,attr"`
	Stop    string     `xml:"stop,attr"`
	Channel string     `xml:"channel,attr"`
	Title   XMLTVText  `xml:"title"`
	Desc    *XMLTVText `xml:"desc,omitempty"`
	Icon    *XMLTVIcon `xml:"icon,omitempty"`
}

// DVBEITEvent represents a DVB-SI Event Information Table entry
type DVBEITEvent struct {
	EventID         uint16    `json:"event_id"`
	StartTime       time.Time `json:"start_time"`
	DurationSeconds int       `json:"duration_seconds"`
	RunningStatus   uint8     `json:"running_status"` // 1=not running, 2=starts in few seconds, 3=pausing, 4=running
	FreeCAMode      bool      `json:"free_ca_mode"`   // 0=free to air, 1=scrambled
	EventName       string    `json:"event_name"`
	ShortDesc       string    `json:"short_desc"`
}

// DVBEITTable represents DVB-EIT section metadata
type DVBEITTable struct {
	ServiceID            uint16        `json:"service_id"`
	TransportStreamID    uint16        `json:"transport_stream_id"`
	OriginalNetworkID    uint16        `json:"original_network_id"`
	VersionNumber        uint8         `json:"version_number"`
	CurrentNextIndicator bool          `json:"current_next_indicator"`
	Events               []DVBEITEvent `json:"events"`
}

// Generator produces XMLTV and DVB-EIT structures
type Generator struct{}

// NewGenerator creates a new EPG generator
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateXMLTV outputs standard XMLTV XML for the given channel and scheduled items
func (g *Generator) GenerateXMLTV(channel models.Channel, items []models.ScheduleItem) ([]byte, error) {
	root := XMLTVRoot{
		GeneratorInfoName: "MCRFlow Master Control Playout",
		Channels: []XMLTVChannel{
			{
				ID:          channel.ID,
				DisplayName: channel.Name,
				Icon: &XMLTVIcon{
					Src: channel.LogoPath,
				},
			},
		},
		Programmes: make([]XMLTVProgram, 0, len(items)),
	}

	for _, it := range items {
		startStr := it.StartTime.Format("20060102150405 -0700")
		stopStr := it.EndTime.Format("20060102150405 -0700")

		prog := XMLTVProgram{
			Start:   startStr,
			Stop:    stopStr,
			Channel: channel.ID,
			Title: XMLTVText{
				Lang:  "en",
				Value: it.ProgramTitle,
			},
		}

		if it.TmdbOverview != "" {
			prog.Desc = &XMLTVText{
				Lang:  "en",
				Value: it.TmdbOverview,
			}
		}

		if it.TmdbPoster != "" {
			prog.Icon = &XMLTVIcon{
				Src: it.TmdbPoster,
			}
		}

		root.Programmes = append(root.Programmes, prog)
	}

	xmlBytes, err := xml.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XMLTV: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	buf.WriteString("<!DOCTYPE tv SYSTEM \"xmltv.dtd\">\n")
	buf.Write(xmlBytes)

	return buf.Bytes(), nil
}

// GenerateDVBEIT constructs DVB-SI Event Information Table data
func (g *Generator) GenerateDVBEIT(serviceID uint16, items []models.ScheduleItem) *DVBEITTable {
	now := time.Now().UTC()
	eit := &DVBEITTable{
		ServiceID:            serviceID,
		TransportStreamID:    1,
		OriginalNetworkID:    1,
		VersionNumber:        0,
		CurrentNextIndicator: true,
		Events:               make([]DVBEITEvent, 0, len(items)),
	}

	for i, it := range items {
		runningStatus := uint8(1) // not running
		if now.After(it.StartTime) && now.Before(it.EndTime) {
			runningStatus = 4 // currently running
		} else if it.StartTime.After(now) && it.StartTime.Sub(now) < 5*time.Minute {
			runningStatus = 2 // starts in few seconds
		}

		eit.Events = append(eit.Events, DVBEITEvent{
			EventID:         uint16(i + 1),
			StartTime:       it.StartTime,
			DurationSeconds: it.DurationSeconds,
			RunningStatus:   runningStatus,
			FreeCAMode:      false,
			EventName:       it.ProgramTitle,
			ShortDesc:       it.TmdbOverview,
		})
	}

	return eit
}
