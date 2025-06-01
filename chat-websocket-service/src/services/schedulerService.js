const cron = require('node-cron');
const messageService = require('./messageService');
const logger = require('../utils/logger');

class SchedulerService {
  constructor() {
    this.jobs = new Map();
  }

  /**
   * Initialize all scheduled jobs
   */
  init() {
    // Check for messages to delete daily at 2 AM
    this.scheduleJob('message-deletion-check', '0 2 * * *', async () => {
      logger.info('Running message deletion check');
      await this.checkMessagesForDeletion();
    });

    // Create deletion warnings daily at 3 AM
    this.scheduleJob('deletion-warning-creation', '0 3 * * *', async () => {
      logger.info('Creating deletion warnings');
      await this.createDeletionWarnings();
    });

    // Actually delete old messages daily at 4 AM
    this.scheduleJob('message-deletion', '0 4 * * *', async () => {
      logger.info('Running message deletion');
      await this.deleteOldMessages();
    });

    logger.info('Scheduler service initialized');
  }

  /**
   * Schedule a cron job
   */
  scheduleJob(name, schedule, handler) {
    if (this.jobs.has(name)) {
      this.jobs.get(name).stop();
    }

    const job = cron.schedule(schedule, async () => {
      try {
        await handler();
      } catch (error) {
        logger.error(`Error in scheduled job ${name}:`, error);
      }
    });

    this.jobs.set(name, job);
    logger.info(`Scheduled job ${name} with schedule ${schedule}`);
  }

  /**
   * Check for messages that should be deleted
   */
  async checkMessagesForDeletion() {
    try {
      const tasksForDeletion = await messageService.getMessagesForDeletion();
      
      for (const task of tasksForDeletion) {
        const deletionDate = this.calculateDeletionDate(task);
        
        // Create warning if deletion is within a month
        const oneMonthFromNow = new Date();
        oneMonthFromNow.setMonth(oneMonthFromNow.getMonth() + 1);
        
        if (deletionDate <= oneMonthFromNow) {
          await messageService.createDeletionWarning(task.task_id, deletionDate);
          logger.info(`Created deletion warning for task ${task.task_id}`);
        }
      }
    } catch (error) {
      logger.error('Error checking messages for deletion:', error);
    }
  }

  /**
   * Calculate when messages should be deleted
   */
  calculateDeletionDate(task) {
    if (task.completed_at) {
      // 6 months after completion
      const deletionDate = new Date(task.completed_at);
      deletionDate.setMonth(deletionDate.getMonth() + 6);
      return deletionDate;
    } else {
      // 1 month after last activity
      const deletionDate = new Date(task.last_message_date);
      deletionDate.setMonth(deletionDate.getMonth() + 1);
      return deletionDate;
    }
  }

  /**
   * Create deletion warnings for tasks
   */
  async createDeletionWarnings() {
    try {
      const tasksForDeletion = await messageService.getMessagesForDeletion();
      
      for (const task of tasksForDeletion) {
        const deletionDate = this.calculateDeletionDate(task);
        await messageService.createDeletionWarning(task.task_id, deletionDate);
      }
      
      logger.info(`Created deletion warnings for ${tasksForDeletion.length} tasks`);
    } catch (error) {
      logger.error('Error creating deletion warnings:', error);
    }
  }

  /**
   * Delete old messages that have passed their deletion date
   */
  async deleteOldMessages() {
    try {
      const query = `
        SELECT task_id 
        FROM message_deletion_warnings 
        WHERE deletion_scheduled_at <= NOW()
      `;
      
      const { rows: tasksToDelete } = await require('../config/database').query(query);
      
      for (const task of tasksToDelete) {
        const deletedCount = await messageService.deleteOldMessages(task.task_id);
        logger.info(`Deleted ${deletedCount} messages for task ${task.task_id}`);
        
        // Remove the warning record
        await require('../config/database').query(
          'DELETE FROM message_deletion_warnings WHERE task_id = $1',
          [task.task_id]
        );
      }
      
      logger.info(`Completed deletion for ${tasksToDelete.length} tasks`);
    } catch (error) {
      logger.error('Error deleting old messages:', error);
    }
  }

  /**
   * Stop all scheduled jobs
   */
  stop() {
    for (const [name, job] of this.jobs) {
      job.stop();
      logger.info(`Stopped scheduled job ${name}`);
    }
    this.jobs.clear();
  }
}

module.exports = new SchedulerService();