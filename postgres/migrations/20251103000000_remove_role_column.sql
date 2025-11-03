-- Remove user_role column from users and drop associated enum type.

-- Drop index referencing the column before removing it.
DROP INDEX IF EXISTS public.users_user_role_idx;

-- Remove the cached role column; role source of truth moves to Kratos traits.
ALTER TABLE public.users
DROP COLUMN IF EXISTS user_role;

-- Drop the enum type that backed the removed column.
DROP TYPE IF EXISTS public.USER_ROLE;
