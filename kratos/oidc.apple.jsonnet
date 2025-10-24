// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: APACHE-2.0

// Apple OIDC Claims Mapper for Kratos

local claims = std.extVar('claims');

{
  identity: {
    traits: {
      email: claims.email,
      name: {
        first: if std.objectHas(claims, 'name') && std.objectHas(claims.name, 'firstName') then claims.name.firstName else '',
        last: if std.objectHas(claims, 'name') && std.objectHas(claims.name, 'lastName') then claims.name.lastName else '',
      },
      picture: '',
      role: 'none',  // Default role: pending admin approval
    },
    metadata_public: {
      provider: 'apple',
      provider_user_id: claims.sub,
    },
  },
}
