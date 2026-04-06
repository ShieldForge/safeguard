import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { LoggingSection } from "./LoggingSection";
import { makeFormData } from "../../test/helpers";

describe("LoggingSection", () => {
    it("renders section title", () => {
        render(<LoggingSection formData={makeFormData()} onChange={vi.fn()} />);
        expect(screen.getByText("Logging")).toBeInTheDocument();
    });

    it("renders debug checkbox (when section is opened)", async () => {
        const user = userEvent.setup();
        render(<LoggingSection formData={makeFormData()} onChange={vi.fn()} />);
        await user.click(screen.getByRole("button", { name: /Logging/i }));
        expect(screen.getByText("Default Debug Logging:")).toBeInTheDocument();
    });

    it("renders log file input when opened", async () => {
        const user = userEvent.setup();
        render(<LoggingSection formData={makeFormData()} onChange={vi.fn()} />);
        await user.click(screen.getByRole("button", { name: /Logging/i }));
        expect(
            screen.getByPlaceholderText(/safeguard\.log/),
        ).toBeInTheDocument();
    });

    it("renders log rotation fields when opened", async () => {
        const user = userEvent.setup();
        render(<LoggingSection formData={makeFormData()} onChange={vi.fn()} />);
        await user.click(screen.getByRole("button", { name: /Logging/i }));
        expect(screen.getByText("Max Log Size (MB):")).toBeInTheDocument();
        expect(screen.getByText("Max Rotated Backups:")).toBeInTheDocument();
        expect(screen.getByText("Max Log Age (Days):")).toBeInTheDocument();
    });
});
