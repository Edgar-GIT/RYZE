-- Back to one open offer per request: keep the oldest open offer of each
-- request and withdraw the others first.
UPDATE premium_level_2_offers o SET status = 'withdrawn', responded_at = CURRENT_TIMESTAMP
WHERE o.status = 'offered'
  AND EXISTS (
      SELECT 1 FROM premium_level_2_offers older
      WHERE older.request_id = o.request_id AND older.status = 'offered'
        AND (older.offered_at, older.id) < (o.offered_at, o.id)
  );
ALTER TABLE premium_level_2_offers DROP CONSTRAINT uq_premium_level_2_offers_request_trainer;
ALTER TABLE premium_level_2_offers ADD CONSTRAINT uq_premium_level_2_offers_open UNIQUE (open_request_id);
