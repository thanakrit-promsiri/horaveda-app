-- =============================================================================
-- Migration: 001_register_with_serial_tx.sql
-- Description: Atomic PostgreSQL Transaction Function (RPC) สำหรับลงทะเบียนผู้ใช้
--              พร้อมตัดสถานะ Serial Key แบบ Row-Level Locking (FOR UPDATE)
--              เพื่อป้องกัน Race Condition และทำการ Rollback อัตโนมัติหากเกิดข้อผิดพลาด
-- =============================================================================

CREATE OR REPLACE FUNCTION public.register_user_with_serial_tx(
    p_username TEXT,
    p_serial_key TEXT
)
RETURNS TABLE (
    user_id INT,
    key_id INT
)
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_key_id INT;
    v_is_used BOOLEAN;
    v_user_id INT;
BEGIN
    -- 1. ค้นหาและล็อกแถวของ Serial Key (FOR UPDATE) เพื่อป้องกันการใช้งานคีย์ซ้ำพร้อมกัน
    SELECT id, is_used
      INTO v_key_id, v_is_used
      FROM public.serial_keys
     WHERE key_code = p_serial_key
       FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'SERIAL_KEY_NOT_FOUND' USING ERRCODE = 'P0002';
    END IF;

    IF v_is_used THEN
        RAISE EXCEPTION 'SERIAL_KEY_ALREADY_USED' USING ERRCODE = 'P0001';
    END IF;

    -- 2. สร้างข้อมูลผู้ใช้ใหม่ในตาราง public.users
    INSERT INTO public.users (username)
    VALUES (p_username)
    RETURNING id INTO v_user_id;

    -- 3. อัปเดตสถานะ Serial Key ว่าถูกใช้งานแล้วและผูกกับ user_id
    UPDATE public.serial_keys
       SET is_used = TRUE,
           user_id = v_user_id
     WHERE id = v_key_id;

    RETURN QUERY SELECT v_user_id, v_key_id;
END;
$$;
