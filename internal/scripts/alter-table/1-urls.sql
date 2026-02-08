alter table urls
add column alias varchar(100)

ALTER TABLE urls
ADD CONSTRAINT unique_alias UNIQUE (short_code);

-- Click Count Added
ALTER TABLE urls
ADD COLUMN click_count BIGINT DEFAULT 0;