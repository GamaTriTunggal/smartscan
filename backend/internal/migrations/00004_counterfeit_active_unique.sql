-- +goose Up
-- Enforce at most one ACTIVE counterfeit detection per QR code.
--
-- The application's create paths do a check-then-create ("is there an active
-- detection? no -> INSERT one"), which under concurrent scans of the same cloned
-- QR can race and insert duplicate active rows. That desyncs qr_codes.counterfeit_status
-- (resolving one active row flips the QR back to 'valid' while another active row
-- lingers). A partial unique index makes the invariant enforceable so the create
-- paths can rely on ON CONFLICT / unique-violation handling instead of a racy read.

-- First collapse any pre-existing duplicate active detections: keep the most
-- recent active row per qr_code_id, demote the rest to 'resolved' so the unique
-- index can be built.
-- +goose StatementBegin
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY qr_code_id
               ORDER BY created_at DESC NULLS LAST, id DESC
           ) AS rn
    FROM public.counterfeit_detections
    WHERE status = 'active'
)
UPDATE public.counterfeit_detections cd
SET status = 'resolved'
FROM ranked
WHERE cd.id = ranked.id
  AND ranked.rn > 1;
-- +goose StatementEnd

CREATE UNIQUE INDEX IF NOT EXISTS uniq_counterfeit_detection_active
    ON public.counterfeit_detections (qr_code_id)
    WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS uniq_counterfeit_detection_active;
