package in.qeet.notify.model;

import com.fasterxml.jackson.annotation.JsonProperty;

/**
 * Request body for POST /v1/events.
 */
public record TriggerEvent(
        @JsonProperty("name")          String name,
        @JsonProperty("subscriber_id") String subscriberId,
        @JsonProperty("payload")       Object payload
) {}
