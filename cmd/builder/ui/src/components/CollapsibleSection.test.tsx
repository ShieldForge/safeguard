import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CollapsibleSection } from "./CollapsibleSection";

describe("CollapsibleSection", () => {
    it("renders the title", () => {
        render(
            <CollapsibleSection title="My Section">
                <p>Content</p>
            </CollapsibleSection>,
        );
        expect(screen.getByText("My Section")).toBeInTheDocument();
    });

    it("is collapsed by default", () => {
        render(
            <CollapsibleSection title="Section">
                <p>Hidden content</p>
            </CollapsibleSection>,
        );
        expect(screen.queryByText("Hidden content")).not.toBeInTheDocument();
    });

    it("is open when defaultOpen is true", () => {
        render(
            <CollapsibleSection title="Section" defaultOpen>
                <p>Visible content</p>
            </CollapsibleSection>,
        );
        expect(screen.getByText("Visible content")).toBeInTheDocument();
    });

    it("toggles open/closed on click", async () => {
        const user = userEvent.setup();
        render(
            <CollapsibleSection title="Section">
                <p>Toggle content</p>
            </CollapsibleSection>,
        );

        const header = screen.getByRole("button", { name: /Section/i });
        expect(screen.queryByText("Toggle content")).not.toBeInTheDocument();

        await user.click(header);
        expect(screen.getByText("Toggle content")).toBeInTheDocument();

        await user.click(header);
        expect(screen.queryByText("Toggle content")).not.toBeInTheDocument();
    });

    it("renders badge when provided", () => {
        render(
            <CollapsibleSection title="Section" badge="required">
                <p>Content</p>
            </CollapsibleSection>,
        );
        expect(screen.getByText("required")).toBeInTheDocument();
    });

    it("renders description when open", async () => {
        const user = userEvent.setup();
        render(
            <CollapsibleSection title="Section" description="Section description">
                <p>Content</p>
            </CollapsibleSection>,
        );

        await user.click(screen.getByRole("button", { name: /Section/i }));
        expect(screen.getByText("Section description")).toBeInTheDocument();
    });

    it("sets aria-expanded correctly", async () => {
        const user = userEvent.setup();
        render(
            <CollapsibleSection title="Section">
                <p>Content</p>
            </CollapsibleSection>,
        );

        const header = screen.getByRole("button", { name: /Section/i });
        expect(header).toHaveAttribute("aria-expanded", "false");

        await user.click(header);
        expect(header).toHaveAttribute("aria-expanded", "true");
    });
});
