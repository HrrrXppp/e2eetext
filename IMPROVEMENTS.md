# Main

- Chat has admins — #11
- Delete user from chat — #12
- Add user to chat after creating chat — #13
- User can leave chat — #14
- Delete chat — #15
- Rotate chat keys — #16
- Rotate user keys — #17
- Sign message by user key — #18
- XSS — #19
- Disappearing messages (chat TTL) — ensure RDS allows `pg_cron` before migrate — #20

# Client

- Search — #21

# API

- Rate limits — #22

# Problems

- ~~Web Socket is reconnecting every min.~~ ~~Fixed: client sends a~~ `heartbeat` ~~every 30s; server replies with~~ `heartbeat.ack` ~~to keep idle connections alive through proxies/ALB.~~ Not fixed. We have reconnect every 5 min. — #23
- In database user_id, chat_id are without node_id in reference fields now. — #24
