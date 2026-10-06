// Chat's timestamps are stored without a time zone, written in UTC (the
// database session runs in UTC). node-postgres would read them in the time
// zone of the machine running chat, shifting every message time by its
// offset; read them as UTC instead.
function parseUtcTimestamp(value) {
  if (!/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)?$/.test(value)) {
    return new Date(value); // e.g. 'infinity': leave it to Date
  }
  return new Date(value.replace(' ', 'T') + 'Z');
}

module.exports = { parseUtcTimestamp };
