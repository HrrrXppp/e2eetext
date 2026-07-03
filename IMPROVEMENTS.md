# Main
- Chat has admins
- Delete user from chat
- Add user to chat after creating chat.
- User can leave chat
- Delete chat

# Client
- Search

# API

# Problems
- ~~Web Socket is reconnecting every min.~~ ~~Fixed: client sends a `heartbeat` every 30s; server replies with `heartbeat.ack` to keep idle connections alive through proxies/ALB.~~ Not fixed. We have reconnect every 5 min.
- In database user_id, chat_id are without node_id in reference fields now.

