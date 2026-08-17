import { betterAuth } from "better-auth";
import { Pool } from "pg";

const databaseUrl = required("DATABASE_URL");

// search_path is pinned to `auth` and deliberately excludes `public`.
//
// Better Auth issues unqualified queries against `sessions`. There is also a
// `public.sessions` holding the talks scheduled inside an event. If `public` were on the
// search path, an unqualified `sessions` could resolve to the domain table, and a login
// would read or write the schedule. Pinning the path to one schema makes that
// impossible rather than unlikely.
const pool = new Pool({
  connectionString: databaseUrl,
  options: "-c search_path=auth",
});

export const auth = betterAuth({
  database: pool,
  secret: required("BETTER_AUTH_SECRET"),
  baseURL: process.env.BETTER_AUTH_URL ?? "http://localhost:3001",
  basePath: "/api/auth",
  trustedOrigins: (process.env.TRUSTED_ORIGINS ?? "http://localhost:5173")
    .split(",")
    .map((origin) => origin.trim())
    .filter(Boolean),

  // Email and password only. No social provider: the assignment does not ask for one,
  // and every provider added would need a client secret and a redirect URL that a
  // reviewer cannot reproduce on a clean machine.
  emailAndPassword: {
    enabled: true,
    // There is no mail server in this stack, so requiring verification would make every
    // seeded and freshly signed-up account unusable.
    requireEmailVerification: false,
    minPasswordLength: 8,
  },

  // These tables already exist. They are created by backend/migrations, which is the
  // single owner of schema in this project, and Better Auth is pointed at them.
  //
  // Nothing here triggers a migration: Better Auth only touches schema through its
  // opt-in CLI (`generate` / `migrate`), never at boot. That CLI is not a dependency of
  // this service, so there is no path by which it could alter a table.
  //
  // Every mapping below is one of two renames:
  //   modelName - the default `user` is a reserved SQL word
  //   fields    - the defaults are camelCase, and unquoted identifiers fold to
  //               lowercase in Postgres
  //
  // A wrong name here fails at query time as "column does not exist", not at startup,
  // which is why this file is TypeScript: the field keys are typed, so a typo in
  // `emailVerified` is a compile error rather than a broken login.
  user: {
    modelName: "users",
    fields: {
      emailVerified: "email_verified",
      createdAt: "created_at",
      updatedAt: "updated_at",
    },
  },
  session: {
    modelName: "sessions",
    fields: {
      userId: "user_id",
      expiresAt: "expires_at",
      ipAddress: "ip_address",
      userAgent: "user_agent",
      createdAt: "created_at",
      updatedAt: "updated_at",
    },
  },
  account: {
    modelName: "accounts",
    fields: {
      userId: "user_id",
      accountId: "account_id",
      providerId: "provider_id",
      accessToken: "access_token",
      refreshToken: "refresh_token",
      idToken: "id_token",
      accessTokenExpiresAt: "access_token_expires_at",
      refreshTokenExpiresAt: "refresh_token_expires_at",
      createdAt: "created_at",
      updatedAt: "updated_at",
    },
  },
  verification: {
    modelName: "verifications",
    fields: {
      expiresAt: "expires_at",
      createdAt: "created_at",
      updatedAt: "updated_at",
    },
  },
});

function required(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is not set`);
  }
  return value;
}
