import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConnectionAuthSection } from "./ConnectionAuthSection";
import { makeFormData } from "../../test/helpers";

describe("ConnectionAuthSection", () => {
    it("renders connection & auth section title", () => {
        render(
            <ConnectionAuthSection
                formData={makeFormData()}
                onChange={vi.fn()}
            />,
        );
        expect(
            screen.getByText(/Connection.*Authentication/),
        ).toBeInTheDocument();
    });

    it("renders vault address input", () => {
        render(
            <ConnectionAuthSection
                formData={makeFormData()}
                onChange={vi.fn()}
            />,
        );
        expect(
            screen.getByPlaceholderText("http://127.0.0.1:8200"),
        ).toBeInTheDocument();
    });

    it("renders auth method select", () => {
        render(
            <ConnectionAuthSection
                formData={makeFormData()}
                onChange={vi.fn()}
            />,
        );
        expect(screen.getByText("OIDC")).toBeInTheDocument();
        expect(screen.getByText("LDAP")).toBeInTheDocument();
        expect(screen.getByText("Token")).toBeInTheDocument();
    });

    it("shows LDAP fields when auth method is ldap", () => {
        render(
            <ConnectionAuthSection
                formData={makeFormData({ default_auth_method: "ldap" })}
                onChange={vi.fn()}
            />,
        );
        expect(
            screen.getByPlaceholderText("Leave empty for runtime prompt"),
        ).toBeInTheDocument();
        expect(screen.getByText(/Security Note/)).toBeInTheDocument();
    });

    it("hides LDAP fields for other auth methods", () => {
        render(
            <ConnectionAuthSection
                formData={makeFormData({ default_auth_method: "oidc" })}
                onChange={vi.fn()}
            />,
        );
        expect(
            screen.queryByPlaceholderText("Leave empty for runtime prompt"),
        ).not.toBeInTheDocument();
    });

    it("shows token field when auth method is token", () => {
        render(
            <ConnectionAuthSection
                formData={makeFormData({ default_auth_method: "token" })}
                onChange={vi.fn()}
            />,
        );
        expect(
            screen.getByPlaceholderText("Leave empty for security"),
        ).toBeInTheDocument();
    });

    it("calls onChange when vault address is typed", async () => {
        const user = userEvent.setup();
        const onChange = vi.fn();
        render(
            <ConnectionAuthSection formData={makeFormData()} onChange={onChange} />,
        );
        const input = screen.getByPlaceholderText("http://127.0.0.1:8200");
        await user.type(input, "a");
        expect(onChange).toHaveBeenCalled();
    });
});
