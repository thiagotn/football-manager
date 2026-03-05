-- Remove attendance records for super admin players from all matches
-- Super admins (role = 'admin') do not participate in matches
DELETE FROM attendances
WHERE player_id IN (
    SELECT id FROM players WHERE role = 'admin'
);
