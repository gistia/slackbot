DROP TABLE IF EXISTS "settings";
CREATE TABLE "settings" (
  "id" bigserial NOT NULL,
  "user" varchar(255) NOT NULL,
  "name" varchar(255) NOT NULL,
  "value" text NOT NULL,
  "created_at" timestamp default NULL,
  CONSTRAINT settings_pkey PRIMARY KEY (id)
) WITH (OIDS=FALSE);

DROP TABLE IF EXISTS "projects";
CREATE TABLE "projects" (
  "id" bigserial NOT NULL,
  "name" varchar(255) NOT NULL,
  "pivotal_id" int NOT NULL,
  "mavenlink_id" int NOT NULL,
  "created_by" varchar(255) NOT NULL,
  "created_at" timestamp default NULL,
  CONSTRAINT projects_pkey PRIMARY KEY (id),
  CONSTRAINT projects_name UNIQUE ("name")
) WITH (OIDS=FALSE);

ALTER TABLE "projects" ADD "mvn_sprint_story_id" varchar(255);
ALTER TABLE "projects" ADD "channel" varchar(255);

DROP TABLE IF EXISTS "users";
CREATE TABLE "users" (
  "id" bigserial NOT NULL,
  "name" varchar(255) NOT NULL,
  "pivotal_id" int NULL,
  "mavenlink_id" int NULL,
  "created_at" timestamp default CURRENT_TIMESTAMP,
  CONSTRAINT users_pkey PRIMARY KEY (id),
  CONSTRAINT users_name UNIQUE ("name")
) WITH (OIDS=FALSE);

ALTER TABLE ONLY "settings" ALTER COLUMN "created_at"
  SET DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE ONLY "projects" ALTER COLUMN "created_at"
  SET DEFAULT CURRENT_TIMESTAMP;

DROP TABLE IF EXISTS "activities";
CREATE TABLE "activities" (
  "id" bigserial NOT NULL,
  "user" varchar(255) NOT NULL,
  "channel" varchar(255) NOT NULL,
  "task" varchar(255) NOT NULL,
  "token" varchar(255) NOT NULL,
  "created_at" timestamp default NULL,
  CONSTRAINT activities_pkey PRIMARY KEY (id)
) WITH (OIDS=FALSE);

DROP TABLE IF EXISTS "poker_sessions";
CREATE TABLE "poker_sessions" (
  "id" bigserial NOT NULL,
  "channel" varchar(255) NOT NULL,
  "title" varchar(255) NOT NULL,
  "users" varchar(255) NOT NULL,
  "finished_at" timestamp default NULL,
  "created_at" timestamp default CURRENT_TIMESTAMP,
  CONSTRAINT poker_sessions_pkey PRIMARY KEY (id)
) WITH (OIDS=FALSE);

DROP TABLE IF EXISTS "poker_stories";
CREATE TABLE "poker_stories" (
  "id" bigserial NOT NULL,
  "poker_session_id" varchar(255) NOT NULL,
  "title" varchar(255) NOT NULL,
  "estimation" numeric NULL,
  "created_at" timestamp default CURRENT_TIMESTAMP,
  CONSTRAINT poker_stories_pkey PRIMARY KEY (id)
) WITH (OIDS=FALSE);

DROP TABLE IF EXISTS "poker_votes";
CREATE TABLE "poker_votes" (
  "id" bigserial NOT NULL,
  "poker_story_id" varchar(255) NOT NULL,
  "user" varchar(255) NOT NULL,
  "vote" numeric NOT NULL,
  "created_at" timestamp default CURRENT_TIMESTAMP,
  CONSTRAINT poker_votes_pkey PRIMARY KEY (id),
  CONSTRAINT poker_votes_story_user UNIQUE ("poker_story_id", "user")
) WITH (OIDS=FALSE);

DROP TABLE IF EXISTS "timers";
CREATE TABLE "timers" (
  "id" bigserial NOT NULL,
  "user" varchar(255) NOT NULL,
  "name" varchar(255) NOT NULL,
  "finished_at" timestamp default NULL,
  "created_at" timestamp default CURRENT_TIMESTAMP,
  CONSTRAINT timers_pkey PRIMARY KEY (id)
) WITH (OIDS=FALSE);

ALTER TABLE "projects" ADD "channel" varchar(255);
ALTER TABLE "projects" ADD "active" BOOLEAN;
UPDATE "projects" SET "active" = TRUE;
ALTER TABLE "projects" ALTER COLUMN "active" SET NOT NULL;
ALTER TABLE "projects" ALTER COLUMN "active" SET DEFAULT TRUE;
