package ws

import "testing"

func TestIsHeartbeat(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    bool
	}{
		{
			name:    "heartbeat",
			payload: `{"type":"heartbeat"}`,
			want:    true,
		},
		{
			name:    "heartbeat ack",
			payload: `{"type":"heartbeat.ack"}`,
			want:    false,
		},
		{
			name:    "chat unread",
			payload: `{"type":"chat.unread","chats":[]}`,
			want:    false,
		},
		{
			name:    "invalid json",
			payload: "not-json",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHeartbeat([]byte(tt.payload)); got != tt.want {
				t.Fatalf("IsHeartbeat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeartbeatAckPayload(t *testing.T) {
	payload, err := HeartbeatAckPayload()
	if err != nil {
		t.Fatalf("HeartbeatAckPayload() error = %v", err)
	}

	if string(payload) != `{"type":"heartbeat.ack"}` {
		t.Fatalf("HeartbeatAckPayload() = %s", payload)
	}
}
