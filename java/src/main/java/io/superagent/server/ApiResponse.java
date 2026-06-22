package io.superagent.server;

import java.util.Map;

/**
 * Standard API response wrapper matching the Go base format.
 *
 * <p>All v2 endpoints return {@code {"code": 0, "msg": "ok", "data": {...}}}
 * for success, and {@code {"code": <errno>, "msg": "<error>", "data": null}}
 * for errors.</p>
 */
public record ApiResponse(int code, String msg, Object data) {

    /** Success response with data payload. */
    public static ApiResponse ok(Object data) {
        return new ApiResponse(0, "ok", data);
    }

    /** Success response with no data. */
    public static ApiResponse ok() {
        return new ApiResponse(0, "ok", null);
    }

    /** Error response with code and message. */
    public static ApiResponse error(int code, String msg) {
        return new ApiResponse(code, msg, null);
    }

    /** Error response with default code -1. */
    public static ApiResponse error(String msg) {
        return new ApiResponse(-1, msg, null);
    }

    /** Convenience: convert to Map for framework serialization. */
    public Map<String, Object> toMap() {
        return Map.of("code", code, "msg", msg, "data", data != null ? data : Map.of());
    }
}
