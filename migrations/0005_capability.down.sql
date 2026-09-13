DROP TABLE cpt_schedule_cutoffs;
DROP TABLE cpt_schedules;
ALTER TABLE process_paths
    DROP COLUMN cycle_time_p95,
    DROP COLUMN eligibility;
