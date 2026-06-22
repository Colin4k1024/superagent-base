package io.superagent.server;

import org.springframework.http.MediaType;
import org.springframework.http.codec.multipart.FilePart;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Mono;

import java.time.Instant;
import java.util.*;

/**
 * File management endpoints.
 *
 * <p>Provides upload, listing, metadata retrieval, content download,
 * and deletion for files managed by the platform.</p>
 */
@RestController
@RequestMapping("/api/v2/files")
public class FileController {

    /**
     * Upload a file (multipart/form-data).
     */
    @PostMapping(consumes = MediaType.MULTIPART_FORM_DATA_VALUE)
    public Mono<Map<String, Object>> upload(@RequestPart("file") FilePart filePart) {
        String id = UUID.randomUUID().toString();
        Map<String, Object> fileMeta = new LinkedHashMap<>();
        fileMeta.put("id", id);
        fileMeta.put("filename", filePart.filename());
        fileMeta.put("content_type", "application/octet-stream");
        fileMeta.put("size", 0);
        fileMeta.put("created_at", Instant.now().toString());
        return Mono.just(ApiResponse.ok(fileMeta).toMap());
    }

    /**
     * List all files.
     */
    @GetMapping
    public Mono<Map<String, Object>> list(
            @RequestParam(defaultValue = "0") int offset,
            @RequestParam(defaultValue = "20") int limit) {
        return Mono.just(ApiResponse.ok(Map.of(
            "files", List.of(),
            "total", 0
        )).toMap());
    }

    /**
     * Get file metadata by ID.
     */
    @GetMapping("/{id}")
    public Mono<Map<String, Object>> getFile(@PathVariable String id) {
        Map<String, Object> fileMeta = new LinkedHashMap<>();
        fileMeta.put("id", id);
        fileMeta.put("filename", "stub");
        fileMeta.put("content_type", "application/octet-stream");
        fileMeta.put("size", 0);
        fileMeta.put("created_at", Instant.now().toString());
        return Mono.just(ApiResponse.ok(fileMeta).toMap());
    }

    /**
     * Download file content.
     */
    @GetMapping("/{id}/content")
    public Mono<Map<String, Object>> getContent(@PathVariable String id) {
        return Mono.just(ApiResponse.ok(Map.of(
            "id", id,
            "content", ""
        )).toMap());
    }

    /**
     * Delete a file.
     */
    @DeleteMapping("/{id}")
    public Mono<Map<String, Object>> delete(@PathVariable String id) {
        return Mono.just(ApiResponse.ok().toMap());
    }
}
