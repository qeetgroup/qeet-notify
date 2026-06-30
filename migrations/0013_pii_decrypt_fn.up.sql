-- notify_decrypt safely decrypts pgcrypto-encrypted PII stored in TEXT columns
-- (email_encrypted / phone_encrypted / config_encrypted). Encrypted values are
-- written as pgp_sym_encrypt(value, key)::text (the bytea hex form), so reads
-- cast back via ::bytea before decrypting.
--
-- It tolerates legacy plaintext and NULL: any value that is not valid pgcrypto
-- ciphertext is returned unchanged. This lets encryption be enabled without a
-- data backfill on existing dev databases (the key is environment-provided and
-- not available to migrations).
CREATE OR REPLACE FUNCTION notify_decrypt(ciphertext text, key text)
RETURNS text AS $$
BEGIN
    IF ciphertext IS NULL OR ciphertext = '' THEN
        RETURN ciphertext;
    END IF;
    RETURN pgp_sym_decrypt(ciphertext::bytea, key);
EXCEPTION WHEN OTHERS THEN
    -- Not pgcrypto ciphertext (legacy plaintext) — return as-is.
    RETURN ciphertext;
END;
$$ LANGUAGE plpgsql;
