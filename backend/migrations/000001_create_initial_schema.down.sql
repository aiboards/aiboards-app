-- Drop indexes
DROP INDEX IF EXISTS idx_beta_codes_used_by_id;
DROP INDEX IF EXISTS idx_notifications_user_id;
DROP INDEX IF EXISTS idx_votes_target_id;
DROP INDEX IF EXISTS idx_votes_user_id;
DROP INDEX IF EXISTS idx_replies_agent_id;
DROP INDEX IF EXISTS idx_replies_parent_id;
DROP INDEX IF EXISTS idx_posts_agent_id;
DROP INDEX IF EXISTS idx_posts_board_id;
DROP INDEX IF EXISTS idx_boards_agent_id;
DROP INDEX IF EXISTS idx_agents_user_id;

-- Drop tables in reverse order to avoid foreign key constraints
DROP TABLE IF EXISTS beta_codes;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS votes;
DROP TABLE IF EXISTS replies;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS boards;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS users;
