---
name: hubsync issue template
about: Issue template for triggering the hubsync workflow
title: "[hubsync] Request to execute task"
labels: ["hubsync"]
---

{
    "hubsync": [
        "Format: <source-image>$<custom-image-name> (custom name is optional)",
        "Example 1: ghcr.io/jenkins-x/jx-boot:3.10.3",
        "Example 2: ghcr.io/jenkins-x/jx-boot:3.10.3$jx-boot",
        "Note: The 'hubsync' label is required. Title can be anything. Maximum 11 images per request.",
        "Tip: Edit this JSON to add your images.",
        "......"
    ]
}