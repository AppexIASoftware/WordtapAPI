# Course Content Access

## Scope

Published teacher course lessons and their lesson games require a student-specific simulated purchase. Classroom enrollment does not grant that purchase, and leaving a classroom does not remove it. Classroom-specific standalone activities retain their enrollment and audience checks.

## Authorization flow

Lesson detail resolves either ID or slug before applying the same course and lesson policy used by the lesson list and lesson game endpoints. Published teacher lesson details in student course and classroom projections are withheld without an entitlement. Course owners and authorized staff retain their existing access. Platform courses do not require a teacher-course purchase.

Standalone games retain their classroom enrollment and audience checks. In addition, the API loads every `lesson_games` link for the game, regardless of its declared source. The student must be authorized for every linked lesson and, when specified, the originating lesson. Missing relations and lookup failures deny access. A standalone game with no lesson links and no explicit lesson source remains available through its normal classroom audience. A game shared across several courses therefore requires access to all linked lessons; this conservative rule prevents an ambiguous shared game from bypassing a paid course.

## Configuration and storage

Simulated purchases are accepted only when `APP_ENV` is `local`, `development`, `dev`, or `test` and `ENABLE_SIMULATED_COURSE_PURCHASES=true`. They are not payment receipts and cannot grant access when simulation is disabled. They are stored in `simulated_course_purchases`, independently of classroom enrollment. The purchase migration must be applied separately in an isolated non-production database; this change does not run migrations.

## HTTP behavior and failures

`GET /api/v1/lessons/:id` accepts an ID or slug and returns 404 for inaccessible content. `GET /api/v1/courses/:course_id/lessons` and `GET /api/v1/lessons/:id/games` deny unauthorized access. `GET /api/v1/courses/:id` still exposes course metadata, but excludes lesson details until access is granted. Backend errors do not grant access.

Example, with an authenticated student token:

```sh
curl -H "Authorization: Bearer $TOKEN" "$API/api/v1/lessons/lesson-slug"
```

A successful lesson response contains an `items` array; an inaccessible lesson returns `{"error":"No se encontró la lección."}`. Simulated purchase creation and idempotency are documented in `SIMULATED_TEACHER_COURSE_PURCHASES.md`.
