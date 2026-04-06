# Policy: Time-based access control
# This policy restricts access based on time of day

package vault

import future.keywords.if
import future.keywords.in

# Default deny
default result = {"allow": false, "reason": ""}

# Helper: Check if current time is within business hours (9 AM - 5 PM)
is_business_hours {
    time.now_ns() / 1000000000 % 86400 >= 32400  # 9 AM in seconds
    time.now_ns() / 1000000000 % 86400 <= 61200  # 5 PM in seconds
}

# Helper: Check if current day is a weekday (Monday-Friday)
is_weekday {
    day := time.weekday(time.now_ns())
    day >= 1  # Monday
    day <= 5  # Friday
}

# Allow access during business hours on weekdays
result = {"allow": true, "reason": "Business hours access"} {
    is_business_hours
    is_weekday
}

# Allow admin users anytime
result = {"allow": true, "reason": "Admin user (unrestricted)"} {
    input.username == "OBSIDIAN\\admin"
}

# Allow automated processes (by specific PIDs) anytime
result = {"allow": true, "reason": "Scheduled task process"} {
    input.process_name == "scheduled-task.exe"
}
