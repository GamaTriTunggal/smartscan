-- +goose Up
-- Drop schema remnants of removed monetization features (loyalty program,
-- campaigns, static-QR product interactions). None of these columns are read
-- or written by the application anymore.

ALTER TABLE public.products DROP COLUMN IF EXISTS loyalty_points_override;
ALTER TABLE public.qr_codes DROP COLUMN IF EXISTS loyalty_claimed_by;
ALTER TABLE public.qr_codes DROP COLUMN IF EXISTS loyalty_claimed_at;

DROP INDEX IF EXISTS idx_interactions_product_id;
ALTER TABLE public.interactions DROP COLUMN IF EXISTS product_id;

-- Retire the 'campaign' preset type; remaining presets become landing presets.
UPDATE public.theme_presets SET preset_type = 'landing' WHERE preset_type = 'campaign';

-- +goose Down
ALTER TABLE public.products ADD COLUMN IF NOT EXISTS loyalty_points_override bigint;
ALTER TABLE public.qr_codes ADD COLUMN IF NOT EXISTS loyalty_claimed_by uuid;
ALTER TABLE public.qr_codes ADD COLUMN IF NOT EXISTS loyalty_claimed_at timestamp with time zone;

ALTER TABLE public.interactions ADD COLUMN IF NOT EXISTS product_id uuid;
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_interactions_product'
          AND conrelid = 'public.interactions'::regclass
    ) THEN
        ALTER TABLE public.interactions
            ADD CONSTRAINT fk_interactions_product FOREIGN KEY (product_id) REFERENCES public.products(id);
    END IF;
END $$;
-- +goose StatementEnd
CREATE INDEX IF NOT EXISTS idx_interactions_product_id ON public.interactions USING btree (product_id);

-- Note: the preset_type 'campaign' -> 'landing' data change is not reversible.
