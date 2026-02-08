-- Step 1: Create a function
CREATE OR REPLACE FUNCTION notify_url_update() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('url_update', 'changed');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 2: Attach trigger to your table (urls)
CREATE TRIGGER url_update_trigger
AFTER INSERT OR UPDATE OR DELETE ON urls
FOR EACH ROW
EXECUTE PROCEDURE notify_url_update();