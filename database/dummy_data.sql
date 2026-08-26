-- Dummy data for rooms and modules
INSERT INTO rooms (nama, port_prefix, capacity) VALUES
    ('f491',  '21', 5),
    ('f492',  '22', 5),
    ('f4111', '23', 5),
    ('f4112', '24', 5);

INSERT INTO modules (code, name, master_container, lxd_profile) VALUES
    ('netbegin', 'Network Beginner', 'master-netbegin', 'praktikum-netbegin'),
    ('netadmin', 'Network Admin',    'master-netadmin', 'praktikum-netadmin');