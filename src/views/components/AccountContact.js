export default {
    name: 'AccountContact',
    data() {
        return {
            contacts: [],
            contactFilter: 'all', // 'all' or 'chatted'
            loading: false,
        }
    },
    methods: {
        async openModal() {
            this.contactFilter = 'all'; // Reset filter to default when opening
            $('#modalContactList').modal('show');
            await this.refreshContacts();
        },
        async refreshContacts() {
            this.loading = true;
            try {
                this.dtClear();
                await this.submitApi();
                this.dtRebuild();
            } catch (err) {
                showErrorInfo(err);
            } finally {
                this.loading = false;
            }
        },
        async changeFilter(filter) {
            if (this.contactFilter === filter) return;
            this.contactFilter = filter;
            await this.refreshContacts();
        },
        dtClear() {
            const table = $('#account_contacts_table');
            if ($.fn.DataTable.isDataTable(table)) {
                table.DataTable().destroy();
            }
        },
        dtRebuild() {
            // Use a timeout to ensure Vue has rendered the table before DataTables initializes
            setTimeout(() => {
                $('#account_contacts_table').DataTable({
                    "pageLength": 10,
                    "destroy": true,
                }).draw();
            }, 0);
        },
        async submitApi() {
            try {
                const response = await window.http.get(`/user/my/contacts?filter=${this.contactFilter}`);
                this.contacts = response.data.results.data;
                 if (this.contacts.length > 0) {
                    showSuccessInfo("Contatos obtidos");
                } else {
                    showWarningInfo("Nenhum contato encontrado para este filtro.");
                }
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            }
        },
        getPhoneNumber(jid) {
            return jid.split('@')[0];
        },
        exportToCSV() {
            if (!this.contacts || this.contacts.length === 0) {
                showErrorInfo("Nenhum contato para exportar");
                return;
            }
            
            let csvContent = "Número de Telefone,Nome\n";
            this.contacts.forEach(contact => {
                const phoneNumber = this.getPhoneNumber(contact.jid);
                const escapedName = contact.name ? contact.name.replace(/"/g, '""') : "";
                csvContent += `${phoneNumber},"${escapedName}"\n`;
            });
            
            const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.setAttribute('href', url);
            link.setAttribute('download', 'contacts.csv');
            link.style.visibility = 'hidden';
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            
            showSuccessInfo("Contatos exportados para CSV");
        }
    },
    template: `
    <div class="olive card" @click="openModal" style="cursor: pointer">
        <div class="content">
            <a class="ui olive right ribbon label">Contatos</a>
            <div class="header">Meus Contatos</div>
            <div class="description">
                Exibir todos os seus contatos
            </div>
        </div>
    </div>
    
    <!--  Modal Contact List  -->
    <div class="ui large modal" id="modalContactList">
        <i class="close icon"></i>
        <div class="header">
            Meus Contatos
        </div>
        <div class="content">
            <div class="ui buttons" style="margin-bottom: 1em;">
                <button class="ui button" :class="{ 'active positive': contactFilter === 'all' }" @click="changeFilter('all')">Todos</button>
                <div class="or" data-text="ou"></div>
                <button class="ui button" :class="{ 'active positive': contactFilter === 'chatted' }" @click="changeFilter('chatted')">Apenas com Conversa</button>
            </div>
            
            <div v-if="loading" class="ui active centered inline loader" style="margin-top: 2em; margin-bottom: 2em;"></div>

            <table v-show="!loading" class="ui celled table" id="account_contacts_table">
                <thead>
                <tr>
                    <th>Número de Telefone</th>
                    <th>Nome</th>
                </tr>
                </thead>
                <tbody v-if="contacts != null">
                <tr v-for="contact in contacts">
                    <td>{{ getPhoneNumber(contact.jid) }}</td>
                    <td>{{ contact.name }}</td>
                </tr>
                </tbody>
            </table>
        </div>
        <div class="actions">
            <button class="ui green button" @click="exportToCSV">
                <i class="download icon"></i> Exportar para CSV
            </button>
            <div class="ui black deny button">
                Fechar
            </div>
        </div>
    </div>
    `
}
