package ports

import "time"

const (
	EventTypeReady              = "ready"
	EventTypePing               = "ping"
	EventTypeMessagesChanged    = "messages.changed"
	EventTypeAttachmentsChanged = "attachments.changed"
	EventTypeFarewellsChanged   = "farewells.changed"
	EventTypeSettingsChanged    = "settings.changed"
	EventTypeWebhooksChanged    = "webhooks.changed"
)

const (
	EventCodeStreamReady                = "stream.ready"
	EventCodeStreamPing                 = "stream.ping"
	EventCodeMessageCreated             = "message.created"
	EventCodeMessageUpdated             = "message.updated"
	EventCodeMessageDeleted             = "message.deleted"
	EventCodeMessageHeartbeat           = "message.heartbeat"
	EventCodeMessageBulkHeartbeat       = "message.bulk_heartbeat"
	EventCodeMessageAttachmentUploaded  = "message.attachment_uploaded"
	EventCodeMessageAttachmentDeleted   = "message.attachment_deleted"
	EventCodeMessageFarewellCreated     = "message.farewell_created"
	EventCodeMessageFarewellUpdated     = "message.farewell_updated"
	EventCodeMessageFarewellDeleted     = "message.farewell_deleted"
	EventCodeAttachmentUploaded         = "attachment.uploaded"
	EventCodeAttachmentDeleted          = "attachment.deleted"
	EventCodeFarewellCreated            = "farewell.created"
	EventCodeFarewellUpdated            = "farewell.updated"
	EventCodeFarewellDeleted            = "farewell.deleted"
	EventCodeFarewellAttachmentUploaded = "farewell_attachment.uploaded"
	EventCodeFarewellAttachmentDeleted  = "farewell_attachment.deleted"
	EventCodeSettingsSaved              = "settings.saved"
	EventCodeWebhookCreated             = "webhook.created"
	EventCodeWebhookUpdated             = "webhook.updated"
	EventCodeWebhookDeleted             = "webhook.deleted"
)

// RealtimeEvent is delivered to authenticated SSE clients.
type RealtimeEvent struct {
	Type             string            `json:"type"`
	Code             string            `json:"code,omitempty"`
	At               time.Time         `json:"at"`
	Data             map[string]string `json:"data,omitempty"`
	Resource         string            `json:"resource,omitempty"`
	EntityID         string            `json:"entity_id,omitempty"`
	Reason           string            `json:"reason,omitempty"`
	OriginSessionKey string            `json:"-"`
}

// EventStreamPort exposes user-scoped pub/sub for real-time refresh hints.
type EventStreamPort interface {
	Subscribe(userID, clientID, sessionKey string) (<-chan RealtimeEvent, <-chan struct{}, func(), error)
	Publish(userID string, event RealtimeEvent)
}
