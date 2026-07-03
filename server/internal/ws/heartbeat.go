package ws

import "encoding/json"

type heartbeatEvent struct {
	Type string `json:"type"`
}

func IsHeartbeat(payload []byte) bool {
	var event heartbeatEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return false
	}

	return event.Type == EventTypeHeartbeat
}

func HeartbeatAckPayload() ([]byte, error) {
	return json.Marshal(heartbeatEvent{Type: EventTypeHeartbeatAck})
}
