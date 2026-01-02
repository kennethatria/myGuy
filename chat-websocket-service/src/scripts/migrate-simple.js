const { Pool } = require('pg');
const fs = require('fs');
const path = require('path');
const logger = require('../utils/logger');

// Load environment variables
require('dotenv').config();

// Create database pool
const pool = new Pool({
    connectionString: process.env.DATABASE_URL || process.env.DB_CONNECTION
});

/**
 * Simple SQL migration runner
 * Executes migration files in order and tracks them in schema_migrations table
 */
async function runMigrations() {
    const client = await pool.connect();

    try {
        logger.info('Starting database migrations...');

        // Create migrations tracking table
        await client.query(`
            CREATE TABLE IF NOT EXISTS schema_migrations (
                version VARCHAR(255) PRIMARY KEY,
                applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                execution_time_ms INTEGER
            )
        `);
        logger.info('✓ Migration tracking table ready');

        // Read migration files
        const migrationsDir = path.join(__dirname, '../../migrations');

        if (!fs.existsSync(migrationsDir)) {
            throw new Error(`Migrations directory not found: ${migrationsDir}`);
        }

        const files = fs.readdirSync(migrationsDir)
            .filter(f => f.endsWith('.sql'))
            .sort(); // Alphabetical order ensures 001, 002, 003, etc.

        if (files.length === 0) {
            logger.warn('No migration files found');
            return;
        }

        logger.info(`Found ${files.length} migration file(s)`);

        // Execute each migration
        for (const file of files) {
            const version = file.replace('.sql', '');

            // Check if already applied
            const result = await client.query(
                'SELECT 1 FROM schema_migrations WHERE version = $1',
                [version]
            );

            if (result.rows.length > 0) {
                logger.info(`⏭️  Skipping ${file} (already applied)`);
                continue;
            }

            // Read and execute migration
            logger.info(`🔄 Running ${file}...`);
            const sqlPath = path.join(migrationsDir, file);
            const sql = fs.readFileSync(sqlPath, 'utf8');

            const startTime = Date.now();

            try {
                await client.query('BEGIN');

                // Execute the migration SQL
                await client.query(sql);

                // Record migration
                const executionTime = Date.now() - startTime;
                await client.query(
                    'INSERT INTO schema_migrations (version, execution_time_ms) VALUES ($1, $2)',
                    [version, executionTime]
                );

                await client.query('COMMIT');
                logger.info(`✅ Completed ${file} (${executionTime}ms)`);
            } catch (error) {
                await client.query('ROLLBACK');
                logger.error(`❌ Failed ${file}:`, error.message);
                logger.error('Error details:', error);
                throw new Error(`Migration failed: ${file}\n${error.message}`);
            }
        }

        logger.info('✅ All migrations completed successfully');

        // Show migration status
        const migrations = await client.query(
            'SELECT version, applied_at, execution_time_ms FROM schema_migrations ORDER BY applied_at'
        );
        logger.info(`Total migrations applied: ${migrations.rows.length}`);

    } catch (error) {
        logger.error('Migration process failed:', error);
        throw error;
    } finally {
        client.release();
        await pool.end();
    }
}

// Run migrations if this script is called directly
if (require.main === module) {
    runMigrations()
        .then(() => {
            logger.info('Migration process completed');
            process.exit(0);
        })
        .catch((error) => {
            logger.error('Migration error:', error);
            process.exit(1);
        });
}

module.exports = runMigrations;
