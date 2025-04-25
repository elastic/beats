---
navigation_title: "Known issues"

---

# Beats known issues [beats-known-issues]

Known issues are significant defects or limitations that may impact your implementation. These issues are actively being worked on and will be addressed in a future release. Review known issues to help you make informed decisions, such as upgrading to a new version.

% Use the following template to add entries to this page.

% :::{dropdown} Title of known issue
% **Details** 
% On [Month/Day/Year], a known issue was discovered that [description of known issue].

% **Workaround** 
% Workaround description.

% **Resolved**
% On [Month/Day/Year], this issue was resolved.

:::

:::{dropdown} Winlogbeat and Filebeat `winlog` input can crash the Event Log on Windows Server 2025.
**Details** 
On 04/16/2025, a known issue was discovered that can cause a crash of the Event Log service in Windows Server 2025 **when reading forwarded events in an Event Collector setup**. The issue appears for some combinations of filters where the OS handles non-null-terminated strings, leading to the crash.

**Workaround** 
As a workaround, and to prevent crashes, Beats will ignore any filters provided when working with forwarded events on Windows Server 2025 until the issue is resolved.

% **Resolved**
% On [Month/Day/Year], this issue was resolved.