import { describe, it, expect } from "vitest";
import { isURL } from "../utils/helpers";

describe("isURL", () => {
    it("returns true for http URLs", () => {
        expect(isURL("http://example.com")).toBe(true);
    });

    it("returns true for https URLs", () => {
        expect(isURL("https://example.com/policy.rego")).toBe(true);
    });

    it("returns false for local paths", () => {
        expect(isURL("./policies")).toBe(false);
        expect(isURL("/etc/policies")).toBe(false);
    });

    it("returns false for empty string", () => {
        expect(isURL("")).toBe(false);
    });

    it("returns false for non-URL strings", () => {
        expect(isURL("ftp://files.example.com")).toBe(false);
        expect(isURL("just-some-text")).toBe(false);
    });
});
