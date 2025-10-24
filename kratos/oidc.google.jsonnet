// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: APACHE-2.0

// Google OIDC Claims Mapper for Kratos

local claims = std.extVar('claims');

{
  identity: {
    traits: {
      email: claims.email,
      name: {
        first: if std.objectHas(claims, 'given_name') then claims.given_name else '',
        last: if std.objectHas(claims, 'family_name') then claims.family_name else '',
      },
      picture: if std.objectHas(claims, 'picture') then claims.picture else '',
      role: 'none',  // Default role: pending admin approval
    },
    metadata_public: {
      provider: 'google',
      provider_user_id: claims.sub,
    },
  },
}
