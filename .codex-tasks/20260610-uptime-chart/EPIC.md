# Epic: Uptime Chart

## Goal

Add Uptime Chart availability analytics for users and admins.

## Requirements

- Search GitHub for existing sub2api uptime/availability chart support and record findings.
- User view: current user can inspect availability for all traffic, by API key, and by model over 1 hour and 6 hours.
- Admin view: admin can inspect site-wide availability for all traffic, by model, and by group over 1 hour and 6 hours.
- Use existing project patterns and expose failures explicitly.

## Current Findings

- Public GitHub search found existing Sub2API channel monitor work and available-channel monitor features, but no directly reusable Uptime Chart matching the requested 1h/6h user/admin slices.
- Local code has channel monitor views with 7d/15d/30d availability, and usage/ops logging that may be reusable as the aggregation source.
