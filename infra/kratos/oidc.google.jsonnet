// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

local claims = {
  email_verified: false,
} + std.extVar('claims');

{
  identity: {
    traits: {
      [if 'email' in claims && claims.email_verified then 'email' else null]: claims.email,
      [if 'given_name' in claims && 'family_name' in claims then 'name' else null]: {
        first: claims.given_name,
        last: claims.family_name,
      },
      [if 'picture' in claims then 'picture' else null]: claims.picture,
    },
  },
}
