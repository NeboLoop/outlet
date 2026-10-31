-- +goose Up
-- Every send must come from one of the sending brand's own domains (0011's
-- access rules). Brands made before domain identities existed send from a
-- domain SES already verified on the platform account but have no row for
-- it, so their mail (the hub's sign-in mail among it) would stop. Give each
-- brand a row for the domain of its from_email when it has none. The domain
-- is already sending, so it is marked verified; the domain refresh re-checks
-- it against SES like any other.
INSERT INTO domain_identities (id, org_id, domain, verification_status, dkim_status)
SELECT
    lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' ||
    substr(lower(hex(randomblob(2))), 2) || '-' ||
    substr('89ab', 1 + (abs(random()) % 4), 1) || substr(lower(hex(randomblob(2))), 2) || '-' ||
    lower(hex(randomblob(6))),
    o.id,
    lower(substr(trim(o.from_email), instr(trim(o.from_email), '@') + 1)),
    'success',
    'success'
FROM organizations o
WHERE instr(trim(coalesce(o.from_email, '')), '@') > 1
  AND NOT EXISTS (
      SELECT 1 FROM domain_identities d
      WHERE d.org_id = o.id
        AND lower(d.domain) = lower(substr(trim(o.from_email), instr(trim(o.from_email), '@') + 1))
  )
  -- One domain belongs to one brand: never give a brand a domain another
  -- brand already holds.
  AND NOT EXISTS (
      SELECT 1 FROM domain_identities d
      WHERE d.org_id <> o.id
        AND lower(d.domain) = lower(substr(trim(o.from_email), instr(trim(o.from_email), '@') + 1))
  );

-- +goose Down
SELECT 1;
