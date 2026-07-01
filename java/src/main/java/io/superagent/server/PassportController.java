package io.superagent.server;

import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Mono;

import java.time.Instant;
import java.util.Map;

/**
 * Stub Passport authentication controller.
 *
 * <p>Implements the three Coze Studio-compatible auth endpoints used by the
 * web frontend so the Java backend can be used without a separate auth service:
 * <ul>
 *   <li>POST /api/passport/web/email/register/v2/</li>
 *   <li>POST /api/passport/web/email/login/</li>
 *   <li>GET  /api/passport/web/logout/</li>
 * </ul>
 *
 * <p><b>Note:</b> This is a stateless stub for local matrix testing.
 * It accepts any email/password and returns a deterministic mock user.
 * Do NOT use in production.</p>
 */
@RestController
@RequestMapping("/api/passport/web")
public class PassportController {

    private static final long FIXED_USER_ID = 7657359490462777344L;

    @PostMapping("/email/register/v2/")
    public Mono<Map<String, Object>> register(@RequestBody Map<String, Object> body) {
        String email = (String) body.getOrDefault("email", "user@example.com");
        return Mono.just(buildUserResponse(email));
    }

    @PostMapping("/email/login/")
    public Mono<Map<String, Object>> login(@RequestBody Map<String, Object> body) {
        String email = (String) body.getOrDefault("email", "user@example.com");
        return Mono.just(buildUserResponse(email));
    }

    @GetMapping("/logout/")
    public Mono<Map<String, Object>> logout() {
        return Mono.just(Map.of("code", 0, "msg", ""));
    }

    // ── helpers ──────────────────────────────────────────────────────────────

    private Map<String, Object> buildUserResponse(String email) {
        String name = email.contains("@") ? email.substring(0, email.indexOf('@')) : email;
        return Map.of(
            "code", 0,
            "msg",  "",
            "data", Map.of(
                "user_id_str",       Long.toString(FIXED_USER_ID),
                "name",              name,
                "user_unique_name",  name,
                "email",             email,
                "description",       "",
                "avatar_url",        "default_icon/user_default_icon.png",
                "screen_name",       name,
                "app_user_info",     Map.of("user_unique_name", name),
                "locale",            "zh-CN",
                "user_create_time",  Instant.now().getEpochSecond()
            )
        );
    }
}
