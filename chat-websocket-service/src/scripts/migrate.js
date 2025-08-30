const { join } = require('path');
const { spawn } = require('child_process');
const logger = require('../utils/logger');

// Load environment variables
require('dotenv').config();

const runMigration = async () => {
  return new Promise((resolve, reject) => {
    // Get database connection from environment
    const dbConnection = process.env.DB_CONNECTION || 'postgresql://postgres:password@localhost:5432/myguy';
    
    // Construct the migration command
    const migrationCommand = 'node-pg-migrate';
    const args = [
      'up',
      '--migration-file-language', 'sql',
      '--migrations-dir', join(__dirname, '../../migrations'),
      '--ignore-pattern', '.*\\.js$', // Ignore JS files
      '--database-url', dbConnection
    ];

    // Run the migration
    const migration = spawn(migrationCommand, args, {
      stdio: 'inherit',
      shell: true
    });

    migration.on('error', (error) => {
      logger.error('Migration failed:', error);
      reject(error);
    });

    migration.on('exit', (code) => {
      if (code === 0) {
        logger.info('Database migration completed successfully');
        resolve();
      } else {
        logger.error(`Migration process exited with code ${code}`);
        reject(new Error(`Migration failed with code ${code}`));
      }
    });
  });
};

// Run migrations if this script is called directly
if (require.main === module) {
  runMigration()
    .then(() => process.exit(0))
    .catch((error) => {
      logger.error('Migration error:', error);
      process.exit(1);
    });
}

module.exports = runMigration;
