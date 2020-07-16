#include <ibus.h>
#include <string.h>

char * randomFnc() {
    return "random";
}

void initIBus() {
    IBusBus * bus = ibus_bus_new();
    IBusComponent * component = ibus_component_new_varargs("name", "org.freedesktop.IBus.telexStandAlone");
    // IBusEngineDesc * engineDesc = ibus_engine_desc_new_varargs("name", "TelexStandAlone");
    IBusEngineDesc * engineDesc = ibus_engine_desc_new_varargs("name", "TelexStandAlone", "language", "us", NULL);
    ibus_component_add_engine(component, engineDesc);
    ibus_bus_register_component(bus, component);
    GDBusConnection * conn = ibus_bus_get_connection(bus);
    ibus_factory_new(conn);
}