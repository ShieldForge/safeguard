import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AccessControlSection } from "./AccessControlSection";
import { makeFormData } from "../../test/helpers";

describe("AccessControlSection", () => {
    it("renders section title", () => {
        render(
            <AccessControlSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(screen.getByText(/Access Control.*Policies/)).toBeInTheDocument();
    });

    it("renders policy path input when opened", async () => {
        const user = userEvent.setup();
        render(
            <AccessControlSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        await user.click(
            screen.getByRole("button", { name: /Access Control/i }),
        );
        expect(
            screen.getByPlaceholderText(/policies.*https/),
        ).toBeInTheDocument();
    });

    it("shows embed-from-URL checkbox when policy path is a URL", async () => {
        const user = userEvent.setup();
        render(
            <AccessControlSection
                formData={makeFormData({
                    default_policy_path: "https://example.com/policy.rego",
                })}
                onChange={vi.fn()}
            />,
        );
        await user.click(
            screen.getByRole("button", { name: /Access Control/i }),
        );
        expect(
            screen.getByText(/Embed policy from URL/),
        ).toBeInTheDocument();
    });

    it("does not show embed checkbox for local paths", async () => {
        const user = userEvent.setup();
        render(
            <AccessControlSection
                formData={makeFormData({ default_policy_path: "./policies" })}
                onChange={vi.fn()}
            />,
        );
        await user.click(
            screen.getByRole("button", { name: /Access Control/i }),
        );
        expect(
            screen.queryByText(/Embed policy from URL/),
        ).not.toBeInTheDocument();
    });
});
