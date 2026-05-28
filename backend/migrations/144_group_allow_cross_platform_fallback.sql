-- 跨平台 fallback 开关：当本分组无可用账号时，是否允许 fallback_group_id
-- 指向的异平台分组接管请求。默认关闭，需显式开启。

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS allow_cross_platform_fallback BOOLEAN NOT NULL DEFAULT FALSE;
