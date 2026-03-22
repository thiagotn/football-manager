-- Migrate existing phone numbers to E.164 format (+55 prefix for Brazilian numbers)
UPDATE players
SET whatsapp = '+55' || whatsapp
WHERE whatsapp NOT LIKE '+%';
