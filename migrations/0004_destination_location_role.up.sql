ALTER TABLE process_paths ADD COLUMN destination_location_role TEXT
    CHECK (destination_location_role IS NULL OR destination_location_role IN ('Drop', 'WorkCenter', 'Shipping'));
