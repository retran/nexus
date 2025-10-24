// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: APACHE-2.0

// Webhook payload for Gateway to create user in database

function(ctx) {
  identity_id: ctx.identity.id,
  email: ctx.identity.traits.email,
  name: if std.objectHas(ctx.identity.traits, 'name') then {
    first: ctx.identity.traits.name.first,
    last: ctx.identity.traits.name.last,
  } else {},
  picture: if std.objectHas(ctx.identity.traits, 'picture') then ctx.identity.traits.picture else '',
  provider: ctx.identity.metadata_public.provider,
  provider_user_id: ctx.identity.metadata_public.provider_user_id,
}
