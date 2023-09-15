const std = @import("std");
const fmt = std.fmt;
const allocator = std.heap.wasm_allocator;

export fn malloc(length: usize) ?[*]u8 {
    const buff = allocator.alloc(u8, length) catch return null;
    return buff.ptr;
}

pub export fn free(buf: [*]u8, length: usize) void {
    allocator.free(buf[0..length]);
}

extern "app" fn getData(keyPointer: *const []u8) [*:0]u8;

export fn hasPerm(userID: i32, meetingID: i32, permPtr: [*:0]u8) i32 {
    const perm = std.mem.span(permPtr);

    const meeting_user_ids = fetch_key([]i32, allocator, "user", userID, "meeting_user_ids") catch return 0;
    const meeting_user_id = find_meeting_user(meetingID, meeting_user_ids);
    const group_ids = fetch_key([]i32, allocator, "meeting_user", meeting_user_id, "group_ids") catch return 0;
    for (group_ids) |group_id| {
        const perms = fetch_key([][]u8, allocator, "group", group_id, "permissions") catch return 0;
        for (perms) |p| {
            if (std.mem.eql(u8, p, perm)) {
                return 1;
            }
        }
    }
    return 0;
}

fn find_meeting_user(meeting_id: i32, meeting_user_ids_: []i32) i32 {
    for (meeting_user_ids_) |meeting_user_id| {
        const mid = fetch_key(i32, allocator, "meeting_user", meeting_user_id, "meeting_id") catch return -1;
        if (mid == meeting_id) {
            return meeting_user_id;
        }
    }
    return -1;
}

fn fetch_key(comptime T: type, alloc: std.mem.Allocator, collection: []const u8, id: i32, field: []const u8) !T {
    const key = try fmt.allocPrint(alloc, "{s}/{d}/{s}", .{ collection, id, field });
    defer allocator.free(key);

    const data = std.mem.span(getData(&key));
    // TODO: free the memory

    const parsed = try std.json.parseFromSlice(T, alloc, data, .{});
    defer parsed.deinit();

    return parsed.value;
}
