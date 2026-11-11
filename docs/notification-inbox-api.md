# Notification inbox API

The authenticated inbox reads the existing `notifications` table. It neither creates nor delivers notifications. Each operation uses the bearer token's user ID; another user's ID is treated exactly like an unknown ID.

## Endpoints

| Endpoint | Result |
| --- | --- |
| `GET /api/v1/notifications?limit=20&cursor=...` | `{ "success": true, "data": { "items": [], "next_cursor": "..." } }` |
| `GET /api/v1/notifications/unread-count` | `{ "success": true, "data": { "unread_count": 0 } }` |
| `PUT /api/v1/notifications/{id}/read` | `{ "success": true }` for an owned notification, including one already read |

Use `Authorization: Bearer <access-token>`. The first page omits `cursor`; subsequent pages use the returned opaque `next_cursor`, which is omitted on the last page. `limit` defaults to 20 and must be 1–100. Ordering is `created_at DESC, id DESC`, with both values in the cursor to break ties. New entries above a cursor do not shift older pages; this is not a snapshot and deletions remain visible as gaps. Unread count is a separate live query and may change between requests.

For example, `curl -H "Authorization: Bearer $ACCESS_TOKEN" "$API_URL/api/v1/notifications?limit=20"` lists the first page. `curl -X PUT -H "Authorization: Bearer $ACCESS_TOKEN" "$API_URL/api/v1/notifications/$ID/read"` marks one read. Repeated reads do not change the original `read_at`. Invalid pagination or UUID input returns 400, invalid authentication 401, missing or foreign IDs 404, and database errors 500. There is no mark-unread or mark-all action.

## Persistence and deployment

No new schema is introduced: the slice uses `user_id`, `is_read`, `read_at`, `created_at`, and `id` from `notifications`. Existing MySQL configuration and authentication settings apply. No migration is required if that existing table matches the model. Database-backed ordering, concurrency, and deployed schema compatibility have not been tested; MySQL integration verification remains necessary before deployment. The Go service must be restarted to load the new routes.
