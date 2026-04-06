import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Tooltip } from "./Tooltip";

describe("Tooltip", () => {
    it("renders the default ? icon when no children provided", () => {
        render(<Tooltip text="Help text" />);
        expect(screen.getByText("?")).toBeInTheDocument();
    });

    it("renders children when provided", () => {
        render(
            <Tooltip text="Help text">
                <span>Custom trigger</span>
            </Tooltip>,
        );
        expect(screen.getByText("Custom trigger")).toBeInTheDocument();
    });

    it("does not show tooltip bubble by default", () => {
        render(<Tooltip text="Help text" />);
        expect(screen.queryByText("Help text")).not.toBeInTheDocument();
    });

    it("shows tooltip on mouse enter and hides on mouse leave", async () => {
        const user = userEvent.setup();
        render(<Tooltip text="Help text" />);

        const wrapper = screen.getByText("?").closest(".tooltip-wrapper")!;
        await user.hover(wrapper);
        expect(screen.getByText("Help text")).toBeInTheDocument();

        await user.unhover(wrapper);
        expect(screen.queryByText("Help text")).not.toBeInTheDocument();
    });
});
