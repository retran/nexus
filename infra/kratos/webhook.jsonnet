// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

function(ctx) {
  identity_id: ctx.identity.id,
  email: ctx.identity.traits.email,
  name: {
    first: if std.objectHas(ctx.identity.traits, 'name') && std.objectHas(ctx.identity.traits.name, 'first') then ctx.identity.traits.name.first else '',
    last: if std.objectHas(ctx.identity.traits, 'name') && std.objectHas(ctx.identity.traits.name, 'last') then ctx.identity.traits.name.last else '',
  },
  picture: if std.objectHas(ctx.identity.traits, 'picture') then ctx.identity.traits.picture else '',
  flow_type: ctx.flow.type,
  flow_id: ctx.flow.id,
}
