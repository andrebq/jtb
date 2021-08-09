// the test is configured to anchor any imports at the "imports" directory and
// shouldNeverBeImported.js is kept in the parent directory that is expected to be off limits;
let shouldNotWork = require("../shouldNeverBeImported.js");
exports.containementFailure = shouldNotWork;
