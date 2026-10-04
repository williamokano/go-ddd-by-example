#!/bin/sh
# Recreates the reference-solution tags (one per lesson, one per Part) on the
# commits of the `solution` branch, then pushes them. Run once from a clone:
#
#   git fetch origin solution && sh scripts/solution-tags.sh && git push origin --tags
#
# New lines are appended as the solution grows; existing tags are left alone.
set -e
tag() { git rev-parse -q --verify "refs/tags/$1" >/dev/null || git tag -a "$1" "$2" -m "$3"; }
tag lesson-0.1   78cac9233daa "Lesson 0.1 — Event storming on paper"
tag lesson-0.2   21c9e8080313 "Lesson 0.2 — Read the reference chapters"
tag lesson-0.3   732625311221 "Lesson 0.3 — Go module, Makefile, linter"
tag lesson-0.4   e42f1058c021 "Lesson 0.4 — Working agreement: branches, commits, CI"
tag part-0       e42f1058c021 "Part 0 — Orientation & setup (done)"
tag lesson-1.1   a5eabf46f2aa "Lesson 1.1 — The domain package and a typed ID"
tag lesson-1.2   ac979c6c1802 "Lesson 1.2 — Value object: Address"
tag lesson-1.3   76693fd00621 "Lesson 1.3 — Value objects: SectionCode and Row"
tag lesson-1.4   e2abcb19942f "Lesson 1.4 — Entity: Section"
tag lesson-1.5   8308f8db6fef "Lesson 1.5 — The aggregate root: registering a Venue"
tag lesson-1.6   b9bbf0c29158 "Lesson 1.6 — Behaviour through the root: AddSection and capacity"
tag lesson-1.7   5a422d2804ac "Lesson 1.7 — Lifecycle: Activate and Retire"
tag lesson-1.8   5b1fe67de839 "Lesson 1.8 — Domain events"
tag lesson-1.9   ce591ff023a7 "Lesson 1.9 — Reconstitution and version"
tag lesson-1.10  1340008d6cb6 "Lesson 1.10 — Checkpoint"
tag part-1       1340008d6cb6 "Part 1 — The Venue domain model (done)"
