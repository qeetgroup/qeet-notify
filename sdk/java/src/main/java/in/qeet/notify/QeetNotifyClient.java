package in.qeet.notify;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import in.qeet.notify.model.TriggerEvent;
import okhttp3.*;

import java.io.IOException;
import java.util.UUID;

/**
 * Thread-safe HTTP client for the Qeet Notify REST API.
 *
 * <pre>{@code
 * var client = new QeetNotifyClient("your-api-key");
 * client.trigger("welcome.email", "user-123", Map.of("name", "Alice"));
 * }</pre>
 */
public class QeetNotifyClient implements AutoCloseable {

    private static final String DEFAULT_BASE_URL = "https://notify.api.qeet.in";
    private static final MediaType JSON = MediaType.get("application/json");

    private final String baseUrl;
    private final String apiKey;
    private final OkHttpClient http;
    private final ObjectMapper mapper;

    public QeetNotifyClient(String apiKey) {
        this(apiKey, DEFAULT_BASE_URL);
    }

    public QeetNotifyClient(String apiKey, String baseUrl) {
        this.apiKey  = apiKey;
        this.baseUrl = baseUrl.replaceAll("/$", "");
        this.http    = new OkHttpClient();
        this.mapper  = new ObjectMapper()
                .registerModule(new JavaTimeModule())
                .disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS);
    }

    // ---- Events ----------------------------------------------------------------

    /**
     * Trigger a notification event asynchronously.
     *
     * @param name         workflow trigger name (e.g. "welcome.email")
     * @param subscriberId subscriber external ID
     * @param payload      arbitrary key-value payload
     * @return raw JSON response body
     */
    public String trigger(String name, String subscriberId, Object payload) throws IOException {
        var event = new TriggerEvent(name, subscriberId, payload);
        var body  = RequestBody.create(mapper.writeValueAsString(event), JSON);
        var req   = new Request.Builder()
                .url(baseUrl + "/v1/events")
                .post(body)
                .header("X-Qeet-Api-Key",  apiKey)
                .header("Idempotency-Key", UUID.randomUUID().toString())
                .build();

        try (var resp = http.newCall(req).execute()) {
            if (!resp.isSuccessful()) {
                throw new IOException("Unexpected response " + resp.code() + ": " + resp.body().string());
            }
            return resp.body().string();
        }
    }

    @Override
    public void close() {
        http.dispatcher().executorService().shutdown();
        http.connectionPool().evictAll();
    }
}
