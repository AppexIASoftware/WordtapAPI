# Notification preferences API

Authenticated users can read and replace their own reminder preferences. This slice stores configuration only; it does not schedule or deliver notifications.

## Schema preparation

The executable, additive MySQL artifact is [`sql/notification_schedule_modes.sql`](sql/notification_schedule_modes.sql). It has **not** been applied or tested against MySQL. Before applying it, back up the database, verify `users.id` is `varchar(36)` with character set and collation matching the database defaults, and verify the table and constraint names are unused. Test application and rollback on a disposable database first. The script intentionally fails if the table exists rather than modifying it. Rollback requires stopping writers and dropping only the new table; doing so loses explicitly selected modes but leaves existing preferences and schedules untouched.

Do not use `AUTO_MIGRATE` as a substitute for this reviewed migration. No new environment variables are required; existing MySQL and authentication configuration applies.

## Contract

`GET /api/v1/notification-preferences` and `PUT /api/v1/notification-preferences` require a bearer access token. The identity comes solely from the token; user IDs are not accepted in the body. Successful responses use `{ "success": true, "data": { ... } }`.

```sh
curl -X PUT "$API_URL/api/v1/notification-preferences" \
  -H "Authorization: Bearer $ACCESS_TOKEN" -H 'Content-Type: application/json' \
  -d '{"enabled":false,"daily_word_enabled":true,"reminder_enabled":true,"timezone":"America/New_York","mode":"custom","schedules":[{"time":"09:30","days":[1,2,3,4,5],"enabled":true}]}'
```

The `data` object returned by GET can be submitted as the PUT body for schedules that satisfy the custom-mode validation below. All three preference booleans, timezone, and mode are required. Timezones must be valid IANA names or `UTC`. Custom mode requires one to five distinct `HH:MM` times; days are distinct integers 1 (Monday) through 7 (Sunday). Every supplied schedule must explicitly contain a non-null `enabled` boolean. Custom PUT replaces reminder schedules only. In random mode, supplied schedules are ignored, not saved or deleted; GET returns the retained custom schedules so that its data can round-trip. To change those schedules, submit custom mode. Random parameters are read-only: they may be omitted or sent unchanged (`random_count: 3`, `random_start: "07:00"`, `random_end: "22:00"`); the API does not pick times.

If no explicit mode exists, existing reminder schedules imply custom mode; no schedules imply random mode. Legacy schedules are returned without silently deleting or normalizing their days, enabled flags, or unusual nonzero seconds. Legacy data that cannot be represented by the custom PUT validation must be reviewed before replacement. Existing `reminder_time` is retained but does not determine the mode. Other notification types are never replaced.

Bad input returns 400; missing or invalid authentication returns 401; persistence failures return 500. PUT is idempotent in resulting configuration, not schedule IDs. Preference and schedule changes commit atomically; calls through this API serialize per user by locking the user row. Other writers that do not acquire the same lock are not serialized with this API. Clients must handle concurrent edits as last-writer-wins. Actual MySQL locking, migration, and rollback remain unverified until tested against a disposable MySQL instance.

## Disposable MySQL test

`TestNotificationPreferencesDisposableMySQL` skips unless `WORDTAP_NOTIFICATION_DISPOSABLE_DSN` is explicitly set and `WORDTAP_NOTIFICATION_ALLOW_CREATE_DB=1`. It creates and drops a uniquely named database on that server, applies the SQL artifact there, and exercises false values, mode changes, rollback and concurrent writes. Run it only against an explicitly authorized disposable server with database creation privileges; never point it at an unapproved server. It uses deliberately minimal fixture tables, so deployment-schema compatibility still requires separate verification.
