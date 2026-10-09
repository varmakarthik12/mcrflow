package channel

import (
	"testing"
	"time"

	"mcrflow/internal/models"
	"mcrflow/internal/resolution"
)

func TestChannelStoreLifecycle(t *testing.T) {
	resStore := resolution.NewStore()
	store := NewStore(resStore)

	// List channels
	channels := store.ListChannels()
	if len(channels) < 2 {
		t.Fatalf("expected at least 2 default channels, got %d", len(channels))
	}

	// Verify resolution preset is bound
	ch1, err := store.GetChannel("ch-01")
	if err != nil {
		t.Fatalf("error retrieving ch-01: %v", err)
	}
	if ch1.ResolutionPreset == nil {
		t.Fatalf("expected ResolutionPreset to be populated")
	}
	if ch1.ResolutionPreset.ID != "res-1080i50-pal-hd" {
		t.Errorf("expected 1080i50 preset on ch-01, got %s", ch1.ResolutionPreset.ID)
	}

	// Create channel
	newCh := &models.Channel{
		Name:                 "Action Prime",
		CallSign:             "ACT-PRIME",
		LogicalChannelNumber: 301,
		ResolutionPresetID:   "res-720p50-hd",
	}
	created, err := store.CreateChannel(newCh)
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if created.ID == "" {
		t.Errorf("expected generated ID")
	}

	// Update channel
	created.Name = "Action Prime 4K"
	created.ResolutionPresetID = "res-4k-2160p50-uhd"
	updated, err := store.UpdateChannel(created.ID, created)
	if err != nil {
		t.Fatalf("failed to update channel: %v", err)
	}
	if updated.Name != "Action Prime 4K" {
		t.Errorf("expected updated name")
	}

	// Emergency slate toggle
	status, err := store.TriggerEmergencySlate(created.ID, true)
	if err != nil || status != "EMERGENCY_SLATE" {
		t.Errorf("expected EMERGENCY_SLATE status, got %s", status)
	}
	status, _ = store.TriggerEmergencySlate(created.ID, false)
	if status != "ON_AIR" {
		t.Errorf("expected ON_AIR status after disengage slate, got %s", status)
	}

	// Delete channel
	if err := store.DeleteChannel(created.ID); err != nil {
		t.Fatalf("failed to delete channel: %v", err)
	}
	if _, err := store.GetChannel(created.ID); err != ErrChannelNotFound {
		t.Errorf("expected ErrChannelNotFound after deletion")
	}
}

func TestChannelRedundancyFailover(t *testing.T) {
	resStore := resolution.NewStore()
	store := NewStore(resStore)

	store.RecordAgentHeartbeat("delhi-dc1-primary")
	store.RecordAgentHeartbeat("mumbai-dc2-primary")

	// Check failover immediately - should NOT fail over
	failovers := store.CheckAndTriggerFailovers(200 * time.Millisecond)
	if len(failovers) != 0 {
		t.Errorf("no failover should occur when heartbeat is fresh")
	}

	// Wait 250ms without new heartbeat
	time.Sleep(250 * time.Millisecond)

	// Check failover - primary agent heartbeat expired!
	failovers = store.CheckAndTriggerFailovers(200 * time.Millisecond)
	if len(failovers) == 0 {
		t.Errorf("expected failover trigger when primary agent heartbeat expires")
	}

	ch1, _ := store.GetChannel("ch-01")
	if ch1.Status != "FAILOVER_STANDBY_ACTIVE" {
		t.Errorf("expected FAILOVER_STANDBY_ACTIVE status, got %s", ch1.Status)
	}
}
