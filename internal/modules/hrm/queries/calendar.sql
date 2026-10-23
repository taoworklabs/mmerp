-- name: WorkWeeks :many
SELECT effective_from, off_days FROM hrm.work_weeks WHERE legal_entity_id = $1 ORDER BY effective_from;

-- name: SaveWorkWeek :exec
INSERT INTO hrm.work_weeks (legal_entity_id, effective_from, off_days) VALUES ($1, $2, $3)
ON CONFLICT (legal_entity_id, effective_from) DO UPDATE SET off_days = excluded.off_days;

-- name: DeleteWorkWeek :execrows
DELETE FROM hrm.work_weeks WHERE legal_entity_id = $1 AND effective_from = $2;

-- name: Holidays :many
SELECT date, name FROM hrm.holidays
WHERE legal_entity_id = @legal_entity_id AND date BETWEEN @from_date::date AND @to_date::date ORDER BY date;

-- name: SaveHoliday :exec
INSERT INTO hrm.holidays (legal_entity_id, date, name) VALUES ($1, $2, $3)
ON CONFLICT (legal_entity_id, date) DO UPDATE SET name = excluded.name;

-- name: DeleteHoliday :execrows
DELETE FROM hrm.holidays WHERE legal_entity_id = $1 AND date = $2;
