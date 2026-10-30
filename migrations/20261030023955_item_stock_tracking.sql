ALTER TABLE item.items
    ADD tracking text NOT NULL DEFAULT 'none' CHECK (tracking IN ('none', 'quantity', 'lot', 'serial')),
    ADD allow_negative boolean NOT NULL DEFAULT false,
    ADD CONSTRAINT items_negative_quantity CHECK (NOT allow_negative OR tracking = 'quantity');
