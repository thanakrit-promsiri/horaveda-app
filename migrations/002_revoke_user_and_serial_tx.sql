-- =============================================================================
-- Migration: 002_revoke_user_and_serial_tx.sql
-- Description: Atomic PostgreSQL Transaction Function (RPC) สำหรับถอนการใช้งาน
--              Serial Key (key_code) ออกจากผู้ใช้ และลบข้อมูลผู้ใช้จากตาราง public.users
-- =============================================================================

CREATE OR REPLACE FUNCTION public.revoke_user_and_serial_tx(
    p_username TEXT
)
RETURNS TABLE (
    deleted_user_id INT,
    revoked_key_code TEXT
)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_user_id INT;
BEGIN
    -- 1. ค้นหาและล็อกแถวของ User (FOR UPDATE)
    SELECT id
      INTO v_user_id
      FROM public.users
     WHERE username = p_username
       FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'USER_NOT_FOUND' USING ERRCODE = 'P0002';
    END IF;

    -- 2. ถอนการใช้งาน Serial Key ที่ผูกกับ user_id นี้ (คืนค่า is_used = FALSE, user_id = NULL)
    --    พร้อมคืนรายการ key_code ที่ถูกถอนสิทธิ์ออกไป
    RETURN QUERY
    UPDATE public.serial_keys
       SET is_used = FALSE,
           user_id = NULL
     WHERE user_id = v_user_id
    RETURNING v_user_id, key_code;

    -- 3. ลบผู้ใช้จากตาราง public.users
    DELETE FROM public.users
     WHERE id = v_user_id;
END;
$$;
