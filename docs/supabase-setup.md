# Setting Up Supabase PostgreSQL for JWT Authentication

This guide explains how to set up a Supabase PostgreSQL database for the JWT authentication system.

## Create a Supabase Project

1. Sign up or log in to [Supabase](https://supabase.com/)
2. Click "New Project"
3. Enter your project details:
   - **Name**: Choose a name for your project
   - **Database Password**: Create a strong password (save this for later)
   - **Region**: Choose a region close to your users
4. Click "Create New Project"

## Get Connection Information

After your project is created, you'll need the connection information:

1. In your Supabase dashboard, go to **Project Settings** (gear icon)
2. Select **Database**
3. Look for the **Connection Info** section
4. Note the following information:
   - **Host**: `[project-ref].supabase.co`
   - **Port**: `5432`
   - **Database name**: `postgres` (default)
   - **User**: `postgres` (default super user)
   - **Password**: The password you set during project creation

## Configure Environment Variables

Update your `.env` file with the Supabase connection information:

```
DB_HOST=db.[project-ref].supabase.co
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-database-password
DB_NAME=postgres
```

## Apply Migrations

The application will automatically run migrations when it starts. Alternatively, you can use a migration tool:

```bash
# Install the migrate CLI tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path migrations -database "postgresql://postgres:kx8-wmiNwoRAioVJ4@db.uaiytxsjkrtnjdfnwuej.supabase.co:5432/postgres?sslmode=require" up
```

## Database Schema

The migrations will create the following schema:

### Users Table

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### Default Users

The migration will insert two default users:

1. **Admin User**
   - Username: `admin`
   - Password: `admin123`
   - Role: `admin`

2. **Regular User**
   - Username: `user`
   - Password: `user123`
   - Role: `user`

## Security Considerations

1. **SSL Connection**: The application uses `sslmode=require` to ensure encrypted connections to the database.

2. **Password Security**: All passwords are hashed using Argon2id before storage.

3. **Connection Pooling**: The application implements connection pooling with configurable limits.

4. **Query Timeouts**: All database operations have context timeouts to prevent long-running queries.

5. **Supabase Security Features**:
   - Row-Level Security (RLS) is available for advanced permission control
   - Network restrictions can be configured in Supabase settings
   - Database auditing is available through Supabase logs

## Troubleshooting

### Connection Issues

If you encounter connection problems:

1. Check that your `DB_HOST` includes the `db.` prefix before the project reference
2. Verify SSL is enabled (`sslmode=require` in the connection string)
3. Check if your IP address needs to be allowlisted in Supabase

### Permission Issues

If you encounter permission errors:

1. Make sure you're using the `postgres` superuser for initial setup
2. Verify the SQL used in migrations has the correct privileges
