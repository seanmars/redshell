import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import { defineComponent, h, ref } from 'vue';
import AppTabs from '../AppTabs.vue';
import AppTab from '../AppTab.vue';

function makeHarness(initialActive = 'one') {
  return defineComponent({
    components: { AppTabs, AppTab },
    setup() {
      const active = ref(initialActive);
      return { active };
    },
    render() {
      return h(
        AppTabs,
        {
          active: this.active,
          'onUpdate:active': (id: string) => {
            this.active = id;
          },
        },
        {
          default: () => [
            h(AppTab, { id: 'one', label: 'One' }, { default: () => 'BodyA' }),
            h(AppTab, { id: 'two', label: 'Two' }, { default: () => 'BodyB' }),
          ],
        },
      );
    },
  });
}

describe('AppTabs', () => {
  it('renders only the active tab body', async () => {
    const wrapper = mount(makeHarness('one'));
    await wrapper.vm.$nextTick();
    expect(wrapper.text()).toContain('BodyA');
    expect(wrapper.text()).not.toContain('BodyB');
  });

  it('clicking a tab emits update:active and switches the rendered body', async () => {
    const wrapper = mount(makeHarness('one'));
    await wrapper.vm.$nextTick();
    const buttons = wrapper.findAll('button[role="tab"]');
    expect(buttons).toHaveLength(2);
    await buttons[1]!.trigger('click');
    await wrapper.vm.$nextTick();
    expect(wrapper.text()).toContain('BodyB');
    expect(wrapper.text()).not.toContain('BodyA');
  });

  it('renders aria-selected reflecting the active state', async () => {
    const wrapper = mount(makeHarness('two'));
    await wrapper.vm.$nextTick();
    const buttons = wrapper.findAll('button[role="tab"]');
    expect(buttons[0]!.attributes('aria-selected')).toBe('false');
    expect(buttons[1]!.attributes('aria-selected')).toBe('true');
  });

  it('renders a tablist container', async () => {
    const wrapper = mount(makeHarness('one'));
    await wrapper.vm.$nextTick();
    expect(wrapper.find('[role="tablist"]').exists()).toBe(true);
  });
});
