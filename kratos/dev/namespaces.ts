// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Nexus IAM - Ory Permission Language Configuration
// Defines permission model for role-based access control

class User implements Namespace {}

class Role implements Namespace {
  related: {
    members: User[]
  }
}

class Resource implements Namespace {
  related: {
    viewers: (User | SubjectSet<Role, "members">)[]
    editors: (User | SubjectSet<Role, "members">)[]
    admins: (User | SubjectSet<Role, "members">)[]
  }

  permits = {
    view: (ctx: Context): boolean =>
      this.related.viewers.includes(ctx.subject) ||
      this.related.editors.includes(ctx.subject) ||
      this.related.admins.includes(ctx.subject),

    edit: (ctx: Context): boolean =>
      this.related.editors.includes(ctx.subject) ||
      this.related.admins.includes(ctx.subject),

    delete: (ctx: Context): boolean =>
      this.related.admins.includes(ctx.subject)
  }
}
