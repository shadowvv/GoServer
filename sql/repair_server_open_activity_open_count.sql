-- Stop-service repair for server_open_activity.open_count.
-- Requirements:
--   1. Run against the server database while gateway/http/game nodes are stopped.
--   2. MySQL 8.0+ is required because ROW_NUMBER() is used.
--   3. The script is safe to rerun. The backup table is created only once.
--
-- Ordering rule:
--   Each (open_server_id, activity_id) starts at open_count = 1 and is ordered
--   by open_time, then version as a deterministic tie-breaker.

DELIMITER $$

DROP PROCEDURE IF EXISTS repair_server_open_activity_open_count$$
CREATE PROCEDURE repair_server_open_activity_open_count()
BEGIN
    DECLARE backup_exists INT DEFAULT 0;
    DECLARE column_exists INT DEFAULT 0;
    DECLARE invalid_count INT DEFAULT 0;

    SELECT COUNT(*)
      INTO backup_exists
      FROM information_schema.tables
     WHERE table_schema = DATABASE()
       AND table_name = 'server_open_activity_before_open_count_repair';

    IF backup_exists = 0 THEN
        CREATE TABLE server_open_activity_before_open_count_repair
            LIKE server_open_activity;
        INSERT INTO server_open_activity_before_open_count_repair
        SELECT * FROM server_open_activity;
    END IF;

    SELECT COUNT(*)
      INTO column_exists
      FROM information_schema.columns
     WHERE table_schema = DATABASE()
       AND table_name = 'server_open_activity'
       AND column_name = 'open_count';

    IF column_exists = 0 THEN
        ALTER TABLE server_open_activity
            ADD COLUMN open_count INT NOT NULL DEFAULT 0 COMMENT 'Activity opening sequence';
    END IF;

    DROP TEMPORARY TABLE IF EXISTS tmp_server_open_activity_open_count;
    CREATE TEMPORARY TABLE tmp_server_open_activity_open_count AS
    SELECT activity_id,
           version,
           open_server_id,
           CAST(
               ROW_NUMBER() OVER (
                   PARTITION BY open_server_id, activity_id
                   ORDER BY open_time ASC, version ASC
               ) AS SIGNED
           ) AS open_count
      FROM server_open_activity;

    ALTER TABLE tmp_server_open_activity_open_count
        ADD PRIMARY KEY (activity_id, version, open_server_id);

    START TRANSACTION;

    UPDATE server_open_activity AS activity
    JOIN tmp_server_open_activity_open_count AS repaired
      ON repaired.activity_id = activity.activity_id
     AND repaired.version = activity.version
     AND repaired.open_server_id = activity.open_server_id
       SET activity.open_count = repaired.open_count;

    SELECT COUNT(*)
      INTO invalid_count
      FROM server_open_activity
     WHERE open_count <= 0;

    IF invalid_count > 0 THEN
        ROLLBACK;
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'open_count repair failed: records with open_count <= 0 remain';
    ELSE
        COMMIT;
    END IF;

    SELECT COUNT(*) AS total_records,
           COUNT(DISTINCT open_server_id, activity_id) AS activity_groups,
           MIN(open_count) AS minimum_open_count,
           MAX(open_count) AS maximum_open_count
      FROM server_open_activity;

    SELECT open_server_id,
           activity_id,
           COUNT(*) AS record_count,
           MIN(open_count) AS minimum_open_count,
           MAX(open_count) AS maximum_open_count,
           COUNT(DISTINCT open_count) AS distinct_open_count
      FROM server_open_activity
     GROUP BY open_server_id, activity_id
    HAVING minimum_open_count <> 1
        OR maximum_open_count <> record_count
        OR distinct_open_count <> record_count;
END$$

CALL repair_server_open_activity_open_count()$$
DROP PROCEDURE repair_server_open_activity_open_count$$

DELIMITER ;
